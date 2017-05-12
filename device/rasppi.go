package device

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ssh_helper"
)

const (
	raspiBoot = "p1"
	raspiMain = "p2"
	bootMount = "/tmp/isaax-boot/"
)

// raspberryPi device
type raspberryPi struct {
	*sdFlasher
}

// Configure overrides sdFlasher Configure() method with custom config
func (d *raspberryPi) Configure() error {
	log.WithField("device", "raspi").Debug("Configure")
	job := help.NewBackgroundJob()
	c := config.NewDefault(d.conf.SSH)

	*(c.GetConfigFn(config.Interface)) = *config.NewCallbackFn(configInterface, saveInterface)
	c.AddConfigFn(config.NewCallbackFn(setupSSH, nil))

	go func() {
		defer job.Close()

		if err := d.MountImg(raspiMain); err != nil {
			job.Error(err)
			return
		}
		if err := d.MountBoot(); err != nil {
			job.Error(err)
		}
	}()

	// setup while background process mounting img
	if err := c.Setup(); err != nil {
		return err
	}

	if err := help.WaitJobAndSpin("Waiting", job); err != nil {
		return err
	}

	// write configs that were setup above
	if err := c.Write(); err != nil {
		return err
	}

	if err := d.UnmountImg(); err != nil {
		return err
	}
	if err := d.Flash(); err != nil {
		return err
	}

	return d.Done()
}

// interfaceConfig is a value for raspberryPi for dhcpcd.conf
var interfaceConfig = `

interface %s
	noipv6rs
	static ip_address=%s
	static routers=%s
	static domain_name_servers=%s
`

// configInterface is a custom configInterface method uses interfaceConfig var
func configInterface(storage map[string]interface{}) error {
	device := []string{"eth0", "wlan0"}
	i := config.Interfaces{
		Address: "192.168.0.254",
		Netmask: "255.255.255.0",
		Gateway: "192.168.0.1",
		DNS:     "192.168.0.1",
	}

	if dialogs.YesNoDialog("Would you like to assign static IP address for your device?") {
		fmt.Println("[+] Available network interface: ")
		num := dialogs.SelectOneDialog("Please select a network interface: ", device)
		fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")

		for {
			fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
				i.Address, i.Gateway, i.Netmask, i.DNS)

			if dialogs.YesNoDialog("Change values?") {
				config.SetInterfaces(&i)

				mask, _ := net.IPMask(net.ParseIP(i.Netmask).To4()).Size()

				storage[config.GetConstLiteral(config.Interface)] = fmt.Sprintf(interfaceConfig, device[num], i.Address+"/"+strconv.Itoa(mask), i.Gateway, i.DNS)

				switch device[num] {
				case "eth0":
					fmt.Println("[+]  Ethernet interface configuration was updated")
				case "wlan0":
					fmt.Println("[+]  wifi interface configuration was updated")
				}
			} else {
				break
			}
		}
	}

	return nil
}

// saveInterface is a custom method and it saves Interfaces value into /etc/dhcpcd.conf
func saveInterface(storage map[string]interface{}) error {

	if _, ok := storage[config.GetConstLiteral(config.Interface)]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", constants.MountDir, constants.ISAAX_CONF_DIR, "dhcpcd.conf")
	command := fmt.Sprintf(`echo "%s" >> %s`, storage[config.GetConstLiteral(config.Interface)], fp)

	_, eut, err := ssh.Run(command)
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// setupSSH is enabling ssh server on pi
func setupSSH(storage map[string]interface{}) error {
	if dialogs.YesNoDialog("Would you like to enable SSH server?") {
		ssh, ok := storage["ssh"].(ssh_helper.Util)
		if !ok {
			return errors.New("Cannot get ssh config")
		}
		command := fmt.Sprintf("touch %sssh", bootMount)
		if _, eut, err := ssh.Run(command); err != nil {
			return errors.New(err.Error() + ":" + eut)
		}
	}
	return nil
}

// MountImg is a method to attach image to loop and mount it
func (d *raspberryPi) MountBoot() error {
	log.Debug("Mounting boot partition")
	//check if image is attached?
	// command := fmt.Sprintf("losetup -f -P %s", help.AddPathSuffix("unix", constants.TMP_DIR, d.img))
	// log.WithField("cmd", command).Debug("Attaching image loop device")
	// if err := d.exec(command); err != nil {
	// 	return err
	// }

	log.Debug("Creating tmp folder")
	if err := d.exec(fmt.Sprintf("mkdir -p %s", bootMount)); err != nil {
		return err
	}

	command := fmt.Sprintf("mount -o rw /dev/loop0%s %s", raspiBoot, bootMount)
	log.WithField("cmd", command).Debug("Mounting tmp folder")
	if err := d.exec(command); err != nil {
		return err
	}
	return nil
}

// UnmountImg is a method to unlink image folder and detach image from the loop
func (d *raspberryPi) UnmountBoot() error {
	log.Debug("Unlinking boot folder")
	command := fmt.Sprintf("umount %s", bootMount)
	if err := d.exec(command); err != nil {
		return err
	}

	log.Debug("Detaching and image")
	command = "losetup -D" // -D detaches all loop devices
	if err := d.exec(command); err != nil {
		return err
	}
	return nil
}

func (d *raspberryPi) exec(command string) error {
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error executing: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" {
		log.Debug(strings.TrimSpace(out))
	}
	return nil
}
