package device

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
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
	ip string
}

// Flash - override default flasher cause configure happens after flashing for edison
func (d *edison) Flash() error {

	if err := d.Prepare(); err != nil {
		return err
	}

	if err := d.Write(); err != nil {
		return err
	}

	if err := d.Configure(); err != nil {
		return err
	}

	return d.Done()
}

// Configure method overrides generic flasher
func (d *edison) Configure() error {
	c := config.New(d.conf.SSH)
	c.AddConfigFn(config.Wifi, config.NewCallbackFn(setupWiFi, nil))
	c.AddConfigFn(config.SSH, config.NewCallbackFn(enableEdisonSSH, nil))
	c.AddConfigFn(config.Interface, config.NewCallbackFn(setupInterface, nil))
	c.AddConfigFn("xIotit", config.NewCallbackFn(setupIotit, nil))

	if err := d.getIPAddress(); err != nil {
		log.Error(err)
	}

	if d.ip == "" {
		fmt.Println("[-] Can't configure board without knowing it's IP")
		return nil
	}

	c.StoreValue("ip", d.ip)

	fmt.Println("[+] Copying your id to the board using ssh-copy-id")
	help.ExecStandardStd("ssh-copy-id", []string{"root@" + d.ip}...)
	time.Sleep(time.Second * 4)

	if err := c.Setup(); err != nil {
		return err
	}

	if err := c.Write(); err != nil {
		return err
	}

	return nil
}

func (d *edison) getIPAddress() error {
	// get IP
	choice := dialogs.SelectOneDialog("Choose Edison's usb-ethernet interface: ", []string{"Select from the list of interfaces", "Enter Edison IP manually"})
	fallback := false

	if choice == 0 {
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
				if iface.Ipv4[:4] == "169." || iface.Ipv4 == "192.168.2.2" {
					if runtime.GOOS == windows {
						arrSel[i] = "~" + iface.Name + "~ ip: " + iface.Ipv4
					} else {
						arrSel[i] = "\x1b[34m" + iface.Name + "\x1b[0m"
					}
				}
			}

			i := dialogs.SelectOneDialog("Please chose correct interface: ", arrSel)
			if runtime.GOOS == windows {
				if ifaces[i].Ipv4[:4] == "169." {
					command := fmt.Sprintf(`netsh int ipv4 set address "%s" static 192.168.2.2 255.255.255.0 192.168.2.1 gwmetric=1`, arr[i])
					fmt.Println("[+] NOTE: You need to run this tool as an administrator")
					if out, err := exec.Command(command).CombinedOutput(); err != nil {
						fmt.Println("[-] Error running '", command, "': ", out)
						fallback = true
					}
				} else {
					fmt.Println("IP is already set to 192.168.2.2, skipping configuration")
					fallback = false
				}
			} else {
				fmt.Println("[+] NOTE: You might need to provide your sudo password")
				if out, err := help.ExecSudo(sudo.InputMaskedPassword, nil, "ifconfig", arr[i], "192.168.2.2"); err != nil {
					fmt.Println("[-] Error running 'sudo ifconfig ", arrSel[i], " 192.168.2.2': ", out)
					fallback = true
				}

			}

			d.ip = "192.168.2.15"
		}
	}

	if choice == 1 || fallback {
		if runtime.GOOS != windows {
			fmt.Println("NOTE: You might need to run `sudo ifconfig {interface} " + dialogs.PrintColored("192.168.2.2") + "` in order to access Edison at " + dialogs.PrintColored("192.168.2.15"))
		}
		d.ip = dialogs.GetSingleAnswer("Input Edison board IP Address (default: 192.168.2.15): ", dialogs.IpAddressValidator)
	}

	if err := help.DeleteHost(filepath.Join(help.UserHomeDir(), ".ssh", "known_hosts"), d.ip); err != nil {
		log.Error(err)
	}

	return nil
}

func setupInterface(storage map[string]interface{}) error {
	ip := storage["ip"].(string)
	var ifaces = config.Interfaces{
		Address: "192.168.0.254",
		Netmask: "255.255.255.0",
		Gateway: "192.168.0.1",
		DNS:     "192.168.0.1",
	}

	if err := setEdisonInterfaces(ifaces, ip); err != nil {
		return err
	}

	fmt.Println("[+] Updating Edison help info") // no idea what this one does and why
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

// Set up Interface values
func setEdisonInterfaces(i config.Interfaces, ip string) error {

	if dialogs.YesNoDialog("Would you like to assign static IP wlan address for your board?") {

		// assign static ip
		fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")

		for {
			fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
				i.Address, i.Gateway, i.Netmask, i.DNS)

			if dialogs.YesNoDialog("Change values?") {
				config.AskInterfaceParams(&i)
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
				fmt.Sprintf("echo nameserver %s > /etc/resolv.conf", i.DNS),
			}
			ifaceDown := []string{
				"root@" + ip,
				"-t",
				fmt.Sprint("ifconfig wlan0 down"),
			}

			ifaceUp := []string{
				"-o",
				"StrictHostKeyChecking=no",
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

// enableEdisonSSH is enabling ssh server on edison
func enableEdisonSSH(storage map[string]interface{}) error {
	ip := storage["ip"].(string)
	if dialogs.YesNoDialog("Would you like to enable SSH on the wireless interface?") {
		fmt.Println("[+] Enabling SSH")
		if err := help.ExecStandardStd("ssh", "root@"+ip, "-t", "configure_edison --password"); err != nil {
			return err
		}
	}
	return nil
}

func setupWiFi(storage map[string]interface{}) error {
	ip := storage["ip"].(string)
	if dialogs.YesNoDialog("Would you like to configure WiFi on your board?") {
		fmt.Println("[+] Updating WiFi configuration")
		if err := help.ExecStandardStd("ssh", "root@"+ip, "-t", "configure_edison --wifi"); err != nil {
			return err
		}
	}
	return nil
}

func setupIotit(storage map[string]interface{}) error {
	ip := storage["ip"].(string)
	base := filepath.Join(config.TmpDir, baseConf)
	baseConf := baseFeeds
	help.WriteToFile(baseConf, base)
	fmt.Println("[+] Uploading base configuration file")
	if err := exec.Command("scp", base, fmt.Sprintf("root@%s:%s", ip, help.AddPathSuffix("unix", "/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(base)

	iotdk := filepath.Join(config.TmpDir, iotdkConf)
	iotdkConf := intelIotdk
	help.WriteToFile(iotdkConf, iotdk)
	fmt.Println("[+] Uploading iot dk config file")
	if err := exec.Command("scp", iotdk, fmt.Sprintf("root@%s:%s", ip, help.AddPathSuffix("unix", "/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(iotdk)
	return nil
}
