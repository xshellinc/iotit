package device

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

const (
	baseConf  string = "base-feeds.conf"
	iotdkConf string = "intel-iotdk.conf"

	baseFeeds string = "src/gz all        http://repo.opkg.net/edison/repo/all\n" +
		"src/gz edison     http://repo.opkg.net/edison/repo/edison\n" +
		"src/gz core2-32   http://repo.opkg.net/edison/repo/core2-32\n"

	intelIotdk string = "src intel-all     http://iotdk.intel.com/repos/1.1/iotdk/all\n" +
		"src intel-iotdk   http://iotdk.intel.com/repos/1.1/intelgalactic\n" +
		"src intel-quark   http://iotdk.intel.com/repos/1.1/iotdk/quark\n" +
		"src intel-i586    http://iotdk.intel.com/repos/1.1/iotdk/i586\n" +
		"src intel-x86     http://iotdk.intel.com/repos/1.1/iotdk/x86\n"
)

// Inits vbox, mounts image, copies config files into image, then closes image, copies image into /tmp
// on the host machine, then flashes it onto mounted disk and eject it cleaning up temporary files
func initEdison() error {
	wg := &sync.WaitGroup{}

	ack := dialogs.YesNoDialog("Would you like to flash your device? ")

	if ack {
		vm, _, _, _ := vboxDownloadImage(wg, lib.VBoxTemplateEdison, constants.DEVICE_TYPE_EDISON)

		printWarnMessage()

		for ack {
			ack = !dialogs.YesNoDialog("Please unplug your edison board. Press yes once unpluged? ")
		}

		for {
			script := "flashall.sh"

			args := []string{
				fmt.Sprintf("%s@%s", lib.TemplateUser, lib.TemplateIP),
				"-p",
				lib.TemplateSSHPort,
				constants.TMP_DIR + script,
			}

			if err := help.ExecStandardStd("ssh", args...); err != nil {
				fmt.Println("[-] Cannot find mounted Intel edison device, please mount it manually")

				if !dialogs.YesNoDialog("Press yes once mounted? ") {
					fmt.Println("Exiting with exit status 2 ...")
					os.Exit(2)
				}

				continue
			}

			break
		}

		fmt.Printf("[+] Stopping virtual machine - Name:%s UUID:%s\n", vm.Name, vm.UUID)

		if err := vbox.Stop(vm.UUID); err != nil {
			log.Error(err)
		}

		progress := make(chan bool)
		go func() {
			defer close(progress)
			time.Sleep(120 * time.Second)
		}()

		help.WaitAndSpin("Configuring edison", progress)
		<-progress

	}

	// Config edison
	config := NewSetDevice(constants.DEVICE_TYPE_EDISON)

	help.ExitOnError(config.SetConfig())

	// Info message
	printDoneMessageUsb()

	return nil
}

// Uses ifconfig to setup edison interface to be accessable via 192.168.2.2 ip
func (e *edison) SetConfig() error {
	// get IP

	i := dialogs.SelectOneDialog("Chose the edison's inteface to connect via usb: ", []string{"Default", "Enter IP"})
	fallback := false

	if i == 0 {
		out, err := help.ExecCmd("sh", []string{"-c", "ifconfig | expand | cut -c1-8 | sort | uniq -u | awk -F: '{print $1;}'"})

		if err != nil {
			log.Error(err)
			fmt.Println("[-] ", err.Error())
			fallback = true
		}

		if !fallback {
			arr := strings.Split(out, "\n")

			tmp := -1
			for idx, v := range arr {
				if strings.Contains(v, "usb") || strings.Contains(v, "en") {
					tmp = idx
				}
			}

			arrSel := make([]string, len(arr)-1)
			copy(arrSel, arr)

			if tmp < 0 {
				arr = append(arr, "\x1b[34musb0\x1b[0m")
			} else {
				arrSel[tmp] = "\x1b[34m" + arr[tmp] + "\x1b[0m"
			}

			i = dialogs.SelectOneDialog("Please chose correct interface: ", arrSel)

			if out, err = help.ExecSudo(sudo.InputMaskedPassword, nil, "ifconfig", arr[i], "192.168.2.2"); err != nil {
				fmt.Println("[-] Error running \x1b[34msudo ifconfig ", arrSel[i], " 192.168.2.2\x1b[0m: ", out)
				fallback = true
			}

			e.ip = "192.168.2.15"
		}
	}

	if i == 1 || fallback {
		fmt.Println("NOTE: You might need to run `sudo ifconfig {interface} \x1b[34m192.168.2.2\x1b[0m` in order to access Edison at \x1b[34m192.168.2.15\x1b[0m")
		e.ip = dialogs.GetSingleAnswer("Input Edison board IP Address: ", []dialogs.ValidatorFn{dialogs.IpAddressValidator})
	}

	if err := deleteHost(filepath.Join((os.Getenv("HOME")), ".ssh", "known_hosts"), e.ip); err != nil {
		log.Error(err)
	}

	time.Sleep(time.Second * 4)

	if err := e.SetInterfaces(*ifaces); err != nil {
		return err
	}

	args := []string{
		"root@" + e.ip,
		"-t",
		"sed -i.bak 's/wireless run configure_edison --password first/wireless run `device config user` first/g' /usr/bin/configure_edison",
	}

	if err := help.ExecStandardStd("ssh", args...); err != nil {
		return err
	}

	base := filepath.Join(constants.TMP_DIR, baseConf)
	baseConf := baseFeeds
	help.WriteToFile(baseConf, base)
	if err := exec.Command("scp", base, fmt.Sprintf("root@%s:%s", e.ip, filepath.Join("/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(base)

	iotdk := filepath.Join(constants.TMP_DIR, iotdkConf)
	iotdkConf := intelIotdk
	help.WriteToFile(iotdkConf, iotdk)
	if err := exec.Command("scp", iotdk, fmt.Sprintf("root@%s:%s", e.ip, filepath.Join("/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(iotdk)

	if err := help.ExecStandardStd("ssh", "root@"+e.ip, "-t", "configure_edison --wifi"); err != nil {
		return err
	}

	if err := help.ExecStandardStd("ssh", "root@"+e.ip, "-t", "configure_edison --password"); err != nil {
		return err
	}

	return nil

}

// Set up Interface values
func (e *edison) SetInterfaces(i Interfaces) error {

	if dialogs.YesNoDialog("Would you like to assign static IP wlan address for your device?") {

		// assign static ip
		fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")

		fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t [+] Network:%s\n\t [+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
			i.Address, i.Network, i.Gateway, i.Netmask, i.DNS)

		if dialogs.YesNoDialog("Change values?") {
			setInterfaces(&i)

			args1 := []string{
				"root@" + e.ip,
				"-t",
				fmt.Sprintf("sed -i.bak -e '53 s/.*/ifconfig $IFNAME %s netmask %s/g' /etc/wpa_supplicant/wpa_cli-actions.sh",
					i.Address, i.Netmask),
			}
			args2 := []string{
				"root@" + e.ip,
				"-t",
				fmt.Sprintf("sed -i -e '54i route add default gw %s' /etc/wpa_supplicant/wpa_cli-actions.sh",
					i.Gateway),
			}
			args3 := []string{
				"root@" + e.ip,
				"-t",
				fmt.Sprintf("echo nameserver %s > /etc/%s", i.DNS, e.resolvF),
			}

			if err := help.ExecStandardStd("ssh", args1...); err != nil {
				return err
			}

			if err := help.ExecStandardStd("ssh", args2...); err != nil {
				return err
			}

			if err := help.ExecStandardStd("ssh", args3...); err != nil {
				return err
			}
		}
	}

	return nil
}

func printWarnMessage() {
	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t WARNNING!!  \t\t\t\t\t\t\t\t\t   *")
	fmt.Println("*\t\t IF YOUR EDISON BOARD IS CONNECTED TO YOUR MACHINE, PLEASE DISCONNECT IT!  \t   *")
	fmt.Println("*\t\t PLEASE FOLLOW THE INSTRUCTIONS! \t\t\t\t\t\t   *")
	fmt.Println(strings.Repeat("*", 100))
}
