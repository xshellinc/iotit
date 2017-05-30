package device

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	ip string
}

func (d *edison) PrepareForFlashing() error {
	ack := dialogs.YesNoDialog("Would you like to flash your board? ")
	if !ack {
		return nil
	}

	if runtime.GOOS == windows {
		return d.flashWindows()
	}

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

	help.WaitJobAndSpin("Your Edison board is restarting...", job)
	return nil
}

func (d *edison) flashWindows() error {
	fileName := ""
	filePath := ""
	if fn, fp, err := d.flasher.DownloadImage(); err == nil {
		fileName = fn
		filePath = fp
	} else {
		return err
	}

	fmt.Printf("[+] Extracting %s \n", fileName)
	if !strings.HasSuffix(fileName, ".zip") {
		return nil
	}
	extractedPath := help.GetTempDir() + help.Separator() + strings.TrimSuffix(fileName, ".zip") + help.Separator()
	command := "unzip"
	args := []string{
		"-o",
		filePath,
		"-d",
		extractedPath,
	}
	log.WithField("args", args).Debug("Extracting an image...")
	if out, err := exec.Command(command, args...).CombinedOutput(); err != nil {
		log.WithField("out", out).Error(err)
		fmt.Println("[-] Error extracting image!", out)
		return err
	}

	d.getDFUUtil(extractedPath)

	script := extractedPath + help.Separator() + "flashall.bat"
	log.Debug("Running flashall... ", script)
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = extractedPath
	if err := cmd.Run(); err != nil {
		fmt.Println("[-] Can't find Intel Edison board, please try to re-connect it")
		os.Exit(2)
	}
	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		time.Sleep(120 * time.Second)
	}()

	help.WaitJobAndSpin("Your Edison board is restarting...", job)
	return nil
}

func (d *edison) getDFUUtil(dst string) error {
	url := "https://cdn.isaax.io/isaax-distro/utilities/dfu-util/dfu-util-0.9-win64.zip"

	if help.Exists(dst + "dfu-util.exe") {
		return nil
	}

	wg := &sync.WaitGroup{}
	fileName, bar, err := help.DownloadFromUrlWithAttemptsAsync(url, dst, 5, wg)
	if err != nil {
		return err
	}

	bar.Prefix(fmt.Sprintf("[+] Download %-15s", fileName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	time.Sleep(time.Second)

	log.WithField("dst", dst).Debug("Extracting")
	if out, err := exec.Command("unzip", "-j", "-o", dst+"dfu-util-0.9-win64.zip", "-d", dst).CombinedOutput(); err != nil {
		log.Debug(string(out))
		return err
	}
	return nil
}

func (d *edison) Configure() error {
	err := d.getIPAddress()
	if err != nil {
		log.Error(err)
	}

	if d.ip == "" {
		fmt.Println("[-] Can't configure board without knowing it's IP")
		return nil
	}

	fmt.Println("[+] Copying your id to the board using ssh-copy-id")
	help.ExecStandardStd("ssh-copy-id", []string{"root@" + d.ip}...)
	time.Sleep(time.Second * 4)

	if err := d.setupInterface(); err != nil {
		return err
	}

	d.configBoard()

	fmt.Println("[+] Done")

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

func (d *edison) setupInterface() error {
	var ifaces = config.Interfaces{
		Address: "192.168.0.254",
		Netmask: "255.255.255.0",
		Gateway: "192.168.0.1",
		DNS:     "192.168.0.1",
	}

	if err := setEdisonInterfaces(ifaces, d.ip); err != nil {
		return err
	}
	fmt.Println("[+] Updating Edison help info") // no idea what this one does and why
	args := []string{
		"root@" + d.ip,
		"-t",
		"sed -i.bak 's/wireless run configure_edison --password first/wireless run `device config user` first/g' /usr/bin/configure_edison",
	}

	if err := help.ExecStandardStd("ssh", args...); err != nil {
		return err
	}

	return nil
}

func (d *edison) configBoard() error {
	if dialogs.YesNoDialog("Would you like to configure WiFi on your board?") {
		fmt.Println("[+] Updating WiFi configuration")
		if err := help.ExecStandardStd("ssh", "root@"+d.ip, "-t", "configure_edison --wifi"); err != nil {
			return err
		}
	}
	base := filepath.Join(constants.TMP_DIR, baseConf)
	baseConf := baseFeeds
	help.WriteToFile(baseConf, base)
	fmt.Println("[+] Uploading base configuration file")
	if err := exec.Command("scp", base, fmt.Sprintf("root@%s:%s", d.ip, filepath.Join("/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(base)

	iotdk := filepath.Join(constants.TMP_DIR, iotdkConf)
	iotdkConf := intelIotdk
	help.WriteToFile(iotdkConf, iotdk)
	fmt.Println("[+] Uploading iot dk config file")
	if err := exec.Command("scp", iotdk, fmt.Sprintf("root@%s:%s", d.ip, filepath.Join("/etc", "opkg"))).Run(); err != nil {
		return err
	}
	os.Remove(iotdk)

	if dialogs.YesNoDialog("Would you like to enable SSH on the wireless interface?") {
		fmt.Println("[+] Enabling SSH")
		if err := help.ExecStandardStd("ssh", "root@"+d.ip, "-t", "configure_edison --password"); err != nil {
			return err
		}
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
