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
	"github.com/xshellinc/iotit/lib/constant"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
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

	ack := dialogs.YesNoDialog("[+] Would you like to flash your device? ")

	if ack {
		vm, _, _, _ := vboxDownloadImage(wg, constant.VBOX_TEMPLATE_EDISON, constants.DEVICE_TYPE_EDISON)

		printWarnMessage()

		for ack {
			ack = !dialogs.YesNoDialog("[+] Please unplug your edison board. Press yes once unpluged? ")
		}

		//@todo replce
		for {
			script := "flashall.sh"
			cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", constant.TEMPLATE_USER, constant.TEMPLATE_IP), "-p", constant.TEMPLATE_SSH_PORT, constants.TMP_DIR+script)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println("error running script, rerunning")
				cmd2 := exec.Command("ssh", fmt.Sprintf("%s@%s", constant.TEMPLATE_USER, constant.TEMPLATE_IP), "-p", constant.TEMPLATE_SSH_PORT, "lsusb | grep Intel")
				cmd2.Stderr = os.Stderr
				out, err := cmd2.Output()
				fmt.Println(string(out), err)

				if string(out) == "" {
					fmt.Println("[-] Cannot find mounted Intel edison device, please mount it manually")

					if !dialogs.YesNoDialog("[+] Press yes once mounted? ") {
						fmt.Println("Exiting with exit status 2 ...")
						os.Exit(2)
					}
				}

				continue
			}

			break
		}

		fmt.Printf("[+] Stopping virtual machine - Name:%s UUID:%s\n", vm.Name, vm.UUID)

		err := vbox.Stop(vm.UUID)
		if err != nil {
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

	i := dialogs.SelectOneDialog("[+] Chose the edison's inteface: ", []string{"Default", "Enter IP"})
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

			i = dialogs.SelectOneDialog("[+] Please chose correct interface: ", arrSel)

			if out, err = help.ExecSudo(help.InputMaskedPassword, nil, "ifconfig", arr[i], "192.168.2.2"); err != nil {
				fmt.Println("[-] Error running \x1b[34msudo ifconfig ", arrSel[i], " 192.168.2.2\x1b[0m: ", out)
				fallback = true
			}

			e.ip = "192.168.2.15"
		}
	}

	if i == 1 || fallback {
		fmt.Println("NOTE: You might need to run `sudo ifconfig {interface} \x1b[34m192.168.2.2\x1b[0m` in order to access Edison at \x1b[34m192.168.2.15\x1b[0m")
		e.ip = dialogs.GetSingleAnswer("[+] Input Edison board IP Address: ", []dialogs.ValidatorFn{dialogs.IpAddressValidator})
	}

	if err := deleteHost(filepath.Join((os.Getenv("HOME")), ".ssh", "known_hosts"), e.ip); err != nil {
		log.Error(err)
	}

	if err := e.SetInterfaces(*ifaces); err != nil {
		return err
	}

	// @todo replace with help
	cmd := exec.Command("ssh", "root@"+e.ip, "-t", "sed -i.bak 's/wireless run configure_edison --password first/wireless run `device config user` first/g' /usr/bin/configure_edison")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		return err
	}

	base := filepath.Join(constants.TMP_DIR, baseConf)
	baseConf := baseFeeds
	help.WriteToFile(baseConf, base)
	// @todo replace with help
	cmd = exec.Command("scp", base, fmt.Sprintf("root@%s:%s", e.ip, filepath.Join("/etc", "opkg")))
	err = cmd.Run()
	if err != nil {
		return err
	}
	os.Remove(base)

	iotdk := filepath.Join(constants.TMP_DIR, iotdkConf)
	iotdkConf := intelIotdk
	help.WriteToFile(iotdkConf, iotdk)
	// @todo replace with help
	cmd = exec.Command("scp", iotdk, fmt.Sprintf("root@%s:%s", e.ip, filepath.Join("/etc", "opkg")))
	err = cmd.Run()
	if err != nil {
		return err
	}
	os.Remove(iotdk)

	// @todo replace with help
	cmd = exec.Command("ssh", "root@"+e.ip, "-t", "configure_edison --wifi")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return err
	}

	// @todo replace with help
	cmd = exec.Command("ssh", "root@"+e.ip, "-t", "configure_edison --password")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil

}

// Set up Interface values
func (e *edison) SetInterfaces(i Interfaces) error {
	var (
		answer string
	)

	fmt.Print("[+] Would you like to assign static IP address for your device?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	for {
		// select network interface
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {

			// assign static ip
			prompt := true
			fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")
			for prompt {
				fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t [+] Network:%s\n\t [+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] Dns:%s\n",
					string(i.Address), string(i.Network), string(i.Gateway), string(i.Netmask), string(i.Dns))
				fmt.Print("[+] Change values?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
				fmt.Scanln(&answer)
				if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
					setInterfaces(&i)
					// @todo replace with help
					cmd := exec.Command("ssh", "root@"+e.ip, "-t", fmt.Sprintf("sed -i.bak -e '53 s/.*/ifconfig $IFNAME %s netmask %s/g' /etc/wpa_supplicant/wpa_cli-actions.sh", i.Address, i.Netmask))
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Stdin = os.Stdin
					err := cmd.Run()
					if err != nil {
						return err
					}
					// @todo replace with help
					cmd = exec.Command("ssh", "root@"+e.ip, "-t", fmt.Sprintf("sed -i -e '54i route add default gw %s' /etc/wpa_supplicant/wpa_cli-actions.sh", i.Gateway))
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Stdin = os.Stdin
					err = cmd.Run()
					if err != nil {
						return err
					}
					// @todo replace with help
					cmd = exec.Command("ssh", "root@"+e.ip, "-t", fmt.Sprintf("echo nameserver %s > /etc/%s", i.Dns, e.resolv_f))
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Stdin = os.Stdin
					err = cmd.Run()
					if err != nil {
						return err
					}
				} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
					return nil
				} else {
					fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
				}
			}
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			return nil
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
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
