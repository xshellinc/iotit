package device

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib/constant"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

func initEdison() error {
	wg := &sync.WaitGroup{}

	vm, _, _, _ := vboxDownloadImage(wg, constant.VBOX_TEMPLATE_EDISON, constants.DEVICE_TYPE_EDISON) // @todo check local

	printWarnMessage()
	// 6. flash edison (in VM)
	var answer string
	for {
		fmt.Print("[+] Would you like to flash your device? (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m): ")
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			script := "flashall.sh"
			// @todo replace with help
			cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", constant.TEMPLATE_USER, constant.TEMPLATE_IP), "-p", constant.TEMPLATE_SSH_PORT, constants.TMP_DIR+script)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
			break
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			//do nothing
			break
		} else {
			fmt.Println("[-] Unknown user input. Please enter (y/yes OR n/no)")
		}
	}

	// 7. Stop VM
	fmt.Printf("[+] Stopping virtual machine - Name:%s UUID:%s\n", vm.Name, vm.UUID)

	err := vm.Poweroff()
	if err != nil {
		log.Error(err)
	}

	// 8. config edison
	config := NewSetDevice(constants.DEVICE_TYPE_EDISON)
	err = config.SetConfig()
	help.ExitOnError(err)

	// 9. Info message
	printDoneMessageUsb()

	return nil
}

func printWarnMessage() {
	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t WARNNING!!  \t\t\t\t\t\t\t\t\t   *")
	fmt.Println("*\t\t IF YOUR EDISON BOARD IS CONNECTED TO YOUR MACHINE, PLEASE DISCONNECT IT!  \t   *")
	fmt.Println("*\t\t PLEASE FOLLOW THE INSTRUCTIONS! \t\t\t\t\t\t   *")
	fmt.Println(strings.Repeat("*", 100))
}

func (e *edison) SetConfig() error {
	const (
		base_conf  string = "base-feeds.conf"
		iotdk_conf string = "intel-iotdk.conf"

		base_feeds string = "src/gz all        http://repo.opkg.net/edison/repo/all\n" +
			"src/gz edison     http://repo.opkg.net/edison/repo/edison\n" +
			"src/gz core2-32   http://repo.opkg.net/edison/repo/core2-32\n"

		intel_iotdk string = "src intel-all     http://iotdk.intel.com/repos/1.1/iotdk/all\n" +
			"src intel-iotdk   http://iotdk.intel.com/repos/1.1/intelgalactic\n" +
			"src intel-quark   http://iotdk.intel.com/repos/1.1/iotdk/quark\n" +
			"src intel-i586    http://iotdk.intel.com/repos/1.1/iotdk/i586\n" +
			"src intel-x86     http://iotdk.intel.com/repos/1.1/iotdk/x86\n"
	)

	// get IP
	edisonIp := ""
	fmt.Println("NOTE: You might need to run `sudo ifconfig {interface} \x1b[34m192.168.2.2\x1b[0m` in order to access Edison at \x1b[34m192.168.2.15\x1b[0m")
	time.Sleep(time.Second * 30)
	fmt.Print("[+] Input Edison board IP Address: ")

	fmt.Scanln(&edisonIp)
	re := regexp.MustCompile("(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})")
	match := re.FindAllString(edisonIp, -1)
	edisonIp = e.ip
	for i := range match {
		edisonIp = match[i]
	}

	fmt.Println("[+] Edison IP: " + edisonIp)

	err := deleteHost(filepath.Join((os.Getenv("HOME")), ".ssh", "known_hosts"), edisonIp)
	if err != nil {
		log.Error(err)
	}

	// static ip config
	err = e.SetInterfaces(*ifaces)
	if err != nil {
		return err
	}

	// @todo replace with help
	cmd := exec.Command("ssh", "root@"+edisonIp, "-t", "sed -i.bak 's/wireless run configure_edison --password first/wireless run `device config user` first/g' /usr/bin/configure_edison")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return err
	}

	base := filepath.Join(constants.TMP_DIR, base_conf)
	baseConf := base_feeds
	help.WriteToFile(baseConf, base)
	// @todo replace with help
	cmd = exec.Command("scp", base, fmt.Sprintf("root@%s:%s", edisonIp, filepath.Join("/etc", "opkg")))
	err = cmd.Run()
	if err != nil {
		return err
	}
	os.Remove(base)

	iotdk := filepath.Join(constants.TMP_DIR, iotdk_conf)
	iotdkConf := intel_iotdk
	help.WriteToFile(iotdkConf, iotdk)
	// @todo replace with help
	cmd = exec.Command("scp", iotdk, fmt.Sprintf("root@%s:%s", edisonIp, filepath.Join("/etc", "opkg")))
	err = cmd.Run()
	if err != nil {
		return err
	}
	os.Remove(iotdk)
	// @todo replace with help
	cmd = exec.Command("ssh", "root@"+edisonIp, "-t", "configure_edison --wifi")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return err
	}
	// @todo replace with help
	cmd = exec.Command("ssh", "root@"+edisonIp, "-t", "configure_edison --password")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil

}

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
