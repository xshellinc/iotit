package device

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
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
	windows = "windows"
)

type edison struct {
	*flasher
}

func (d *edison) PrepareForFlashing() error {
	ack := dialogs.YesNoDialog("Would you like to flash your board? ")

	if ack {
		d.flasher.PrepareForFlashing()
		for !dialogs.YesNoDialog("Please unplug your Edison board. Type yes once unpluged.") {
		}

		for {
			script := "flashall.sh"
			args := []string{
				fmt.Sprintf("%s@%s", vbox.VBoxUser, vbox.VBoxIP),
				"-p",
				vbox.VBoxSSHPort,
				constants.TMP_DIR + script,
			}
			if err := help.ExecStandardStd("ssh", args...); err != nil {
				fmt.Println("[-] Can't find Intel Edison board, please try to re-connect it")

				if !dialogs.YesNoDialog("Type yes once connected.") {
					fmt.Println("Exiting with exit status 2 ...")
					os.Exit(2)
				}
				continue
			}
			break
		}

		if err := vbox.Stop(d.vbox.UUID); err != nil {
			log.Error(err)
		}

		job := help.NewBackgroundJob()
		go func() {
			defer job.Close()
			time.Sleep(120 * time.Second)
		}()

		help.WaitJobAndSpin("Configuring Edison", job)
	}

	return nil
}

func (d *edison) Configure() error {
	err := setConfig()
	if err != nil {
		log.Error(err)
	}

	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t WARNNING!!  \t\t\t\t\t\t\t\t\t   *")
	fmt.Println("*\t\t IF YOUR EDISON BOARD IS CONNECTED TO YOUR MACHINE, PLEASE DISCONNECT IT!  \t   *")
	fmt.Println("*\t\t PLEASE FOLLOW THE INSTRUCTIONS! \t\t\t\t\t\t   *")
	fmt.Println(strings.Repeat("*", 100))

	return nil
}

func setConfig() error {
	// get IP
	var ip string

	i := dialogs.SelectOneDialog("Choose Edison's interface to connect to: ", []string{"Select from a list of interfaces", "Enter IP manually"})
	fallback := false

	if i == 0 {
		ifaces, err := help.LocalIfaces()

		if err != nil || len(ifaces) == 0 {
			log.Error(err)
			fmt.Println("[-] ", err.Error())
			fallback = true
		}

		if !fallback {
			arr := make([]string, len(ifaces))
			arrSel := make([]string, len(ifaces))
			fmt.Println("[+] Highlighted interfaces is our heuristic guess")
			for i, iface := range ifaces {
				arr[i] = iface.Name
				arrSel[i] = iface.Name
				if iface.Ipv4[:4] == "169." {
					if runtime.GOOS == windows {
						arrSel[i] = "**" + iface.Name + "**"
					} else {
						arrSel[i] = "\x1b[34m" + iface.Name + "\x1b[0m"
					}
				}
			}

			i = dialogs.SelectOneDialog("Please chose correct interface: ", arrSel)
			if runtime.GOOS == windows {
				// it's either: netsh interface ipv4 add address “Local Area Connection” 192.168.1.2 255.255.255.0
				// or: netsh int ipv4 set address "%interface%" static %IP% %MASK% %GATE% gwmetric=1
				command := fmt.Sprintf(`netsh int ipv4 set address "%s" static 192.168.2.2 255.255.255.0 192.168.2.1 gwmetric=1`, arr[i])
				fmt.Println("[+] NOTE: You need to run this tool as an administrator")
				if out, err := exec.Command(command).CombinedOutput(); err != nil {
					fmt.Println("[-] Error running '", command, "': ", out)
					fallback = true
				}
			} else {
				fmt.Println("[+] NOTE: You might need to provide your sudo password")
				if out, err := help.ExecSudo(sudo.InputMaskedPassword, nil, "ifconfig", arr[i], "192.168.2.2"); err != nil {
					fmt.Println("[-] Error running 'sudo ifconfig ", arrSel[i], " 192.168.2.2': ", out)
					fallback = true
				}

			}

			ip = "192.168.2.15"
		}
	}

	if i == 1 || fallback {
		if runtime.GOOS == windows {
			fmt.Println("NOTE: You might need to run `netsh interface ipv4 add address \"{interface}\" 192.168.2.2 255.255.255.0`")
			fmt.Println("OR")
			fmt.Println("`netsh int ipv4 set address \"{interface}\" static 192.168.2.2 255.255.255.0 192.168.2.1 gwmetric=1`")
		} else {
			fmt.Println("NOTE: You might need to run `sudo ifconfig {interface} " + dialogs.PrintColored("192.168.2.2") + "` in order to access Edison at " + dialogs.PrintColored("192.168.2.15"))
		}
		ip = dialogs.GetSingleAnswer("Input Edison board IP Address: ", dialogs.IpAddressValidator)
	}

	if err := help.DeleteHost(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"), ip); err != nil {
		log.Error(err)
	}

	fmt.Println("[+] Copying board id")
	help.ExecStandardStd("ssh-copy-id", []string{"root@" + ip}...)
	time.Sleep(time.Second * 4)

	if err := setUpInterface(ip); err != nil {
		return err
	}

	return configBoard(ip)

}

func setUpInterface(ip string) error {
	var ifaces = config.Interfaces{
		Address: "192.168.0.254",
		Netmask: "255.255.255.0",
		Gateway: "192.168.0.1",
		DNS:     "192.168.0.1",
	}

	if err := setEdisonInterfaces(ifaces, ip); err != nil {
		return err
	}
	fmt.Println("[+] Updating Edison help info")
	args := []string{
		"root@" + ip,
		"-t",
		"sed -i.bak 's/wireless run configure_edison --password first/wireless run `device config user` first/g' /usr/bin/configure_edison",
	}

	if err := help.ExecStandardStd("ssh", args...); err != nil {
		return err
	}

	return nil
}

func configBoard(ip string) error {
	base := filepath.Join(constants.TMP_DIR, baseConf)
	baseConf := baseFeeds
	help.WriteToFile(baseConf, base)
	fmt.Println("[+] Uploading base configuration file")
	if err := exec.Command("scp", base, fmt.Sprintf("root@%s:%s", ip, filepath.Join("/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(base)

	iotdk := filepath.Join(constants.TMP_DIR, iotdkConf)
	iotdkConf := intelIotdk
	help.WriteToFile(iotdkConf, iotdk)
	fmt.Println("[+] Uploading iot dk config file")
	if err := exec.Command("scp", iotdk, fmt.Sprintf("root@%s:%s", ip, filepath.Join("/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(iotdk)
	fmt.Println("[+] Updating user password")
	if err := help.ExecStandardStd("ssh", "root@"+ip, "-t", "configure_edison --password"); err != nil {
		return err
	}

	return nil
}

// Set up Interface values
func setEdisonInterfaces(i config.Interfaces, ip string) error {

	if dialogs.YesNoDialog("Would you like to assign static IP wlan address for your board?") {

		// assign static ip
		fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")

		for {
			fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
				i.Address, i.Gateway, i.Netmask, i.DNS)

			if dialogs.YesNoDialog("Change values?") {
				config.SetInterfaces(&i)
			}

			fmt.Println("[+] NOTE: You might need to enter your Edison board password")

			args1 := []string{
				"root@" + ip,
				"-t",
				fmt.Sprintf("sed -i.bak -e '53 s/.*/ifconfig $IFNAME %s netmask %s/g' /etc/wpa_supplicant/wpa_cli-actions.sh",
					i.Address, i.Netmask),
			}

			args2 := []string{
				"root@" + ip,
				"-t",
				fmt.Sprintf("sed -i -e '54i route add default gw %s' /etc/wpa_supplicant/wpa_cli-actions.sh",
					i.Gateway),
			}

			args3 := []string{
				"root@" + ip,
				"-t",
				fmt.Sprintf("echo nameserver %s > /etc/%s", i.DNS, constants.ResolveF),
			}

			ifaceDown := []string{
				"root@" + ip,
				"-t",
				fmt.Sprint("ifconfig wlan0 down"),
			}

			ifaceUp := []string{
				"root@" + ip,
				"-t",
				fmt.Sprint("ifconfig wlan0 up"),
			}

			fmt.Println("[+] Updating network configuration")
			if err := help.ExecStandardStd("ssh", args1...); err != nil {
				return err
			}
			fmt.Println("[+] Updating gateway settings")
			if err := help.ExecStandardStd("ssh", args2...); err != nil {
				return err
			}
			fmt.Println("[+] Adding custom nameserver")
			if err := help.ExecStandardStd("ssh", args3...); err != nil {
				return err
			}
			fmt.Println("[+] Updating WiFi configuration")
			if err := help.ExecStandardStd("ssh", "root@"+ip, "-t", "configure_edison --wifi"); err != nil {
				return err
			}
			fmt.Println("[+] Reloading interface settings")
			if err := help.ExecStandardStd("ssh", ifaceDown...); err != nil {
				fmt.Println("[-] Error shutting down wlan0 interface: ", err.Error())
				return err
			}
			time.Sleep(1 * time.Second)
			if err := help.ExecStandardStd("ssh", ifaceUp...); err != nil {
				fmt.Println("[-] Error starting wlan0 interface: ", err.Error())
				return err
			}
			break
		}

	}
	return nil
}
