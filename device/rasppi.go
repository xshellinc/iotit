package device

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
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

func (d *raspberryPi) prepare() error {
	return d.Prepare()
}

// Configure overrides sdFlasher Configure() method with custom config
func (d *raspberryPi) Configure() error {
	log.WithField("device", "raspi").Debug("Configure")

	job := help.NewBackgroundJob()
	c := config.NewDefault(d.conf.SSH) // create config with default callbacks
	// replace default interface configuration with custom raspi configurator
	c.SetConfigFn(config.Interface, config.NewCallbackFn(setInterface, saveInterface))
	c.AddConfigFn(config.SSH, config.NewCallbackFn(enablePiSSH, nil))
	c.AddConfigFn(config.Camera, config.NewCallbackFn(enablePiCamera, nil))

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

	if err := help.WaitJobAndSpin("Waiting", job); err != nil {
		return err
	}

	fmt.Println("[+] Configuring...")
	if !d.Quiet {
		if dialogs.YesNoDialog("Would you like to configure your board?") {
			if err := c.Setup(); err != nil {
				return err
			}
		}
	} else {
		if err := touchSSH(d.conf.SSH); err != nil {
			fmt.Println("[-] Error:", err.Error())
		}
	}

	// write configs that were setup above
	if err := c.Write(); err != nil {
		return err
	}

	if err := d.UnmountImg(); err != nil {
		return err
	}

	if err := d.UnmountBoot(); err != nil {
		return err
	}

	fmt.Println("[+] Image configured")
	return nil
}

// Flash configures and flashes image
func (d *raspberryPi) Flash() error {

	if err := d.prepare(); err != nil {
		return err
	}

	if err := d.Configure(); err != nil {
		return err
	}

	if err := d.Write(); err != nil {
		return err
	}

	return nil
}

// interfaceConfig is a value for raspberryPi for dhcpcd.conf
var interfaceConfig = `

interface %s
	noipv6rs
	static ip_address=%s
	static routers=%s
	static domain_name_servers=%s
`

// SetInterface is a custom SetInterface method uses interfaceConfig var
func setInterface(storage map[string]interface{}) error {
	log.WithField("type", "raspi").Debug("SetInterface")
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

		fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
			i.Address, i.Gateway, i.Netmask, i.DNS)

		if dialogs.YesNoDialog("Change values?") {
			config.AskInterfaceParams(&i)
		}

		mask, _ := net.IPMask(net.ParseIP(i.Netmask).To4()).Size()

		storage[config.Interface] = fmt.Sprintf(interfaceConfig, device[num], i.Address+"/"+strconv.Itoa(mask), i.Gateway, i.DNS)

		switch device[num] {
		case "eth0":
			fmt.Println("[+]  Ethernet interface configuration was updated")
		case "wlan0":
			fmt.Println("[+]  wifi interface configuration was updated")
		}

	}

	return nil
}

// saveInterface is a custom method and it saves Interfaces value into /etc/dhcpcd.conf
func saveInterface(storage map[string]interface{}) error {
	log.WithField("type", "raspi").Debug("saveInterface")

	if _, ok := storage[config.Interface]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", config.MountDir, config.IsaaxConfDir, "dhcpcd.conf")
	command := fmt.Sprintf(`echo "%s" >> %s`, storage[config.Interface], fp)
	log.WithField("type", "raspi").WithField("command", command).Debug("save interface")
	_, eut, err := ssh.Run(command)
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}
	return nil
}

// enablePiSSH is enabling ssh server on pi
func enablePiSSH(storage map[string]interface{}) error {
	if dialogs.YesNoDialog("Would you like to enable SSH server?") {
		ssh, ok := storage["ssh"].(ssh_helper.Util)
		if !ok {
			return errors.New("Cannot get ssh config")
		}
		return touchSSH(ssh)
	}
	return nil
}

func touchSSH(ssh ssh_helper.Util) error {
	fmt.Println("[+] Enabled SSH server.")
	command := fmt.Sprintf("touch %sssh", bootMount)
	log.WithField("cmd", command).Debug("Enabling SSH")
	if _, eut, err := ssh.Run(command); err != nil {
		return errors.New(err.Error() + ":" + eut)
	}
	return nil
}

// enablePiCamera is enabling camera
func enablePiCamera(storage map[string]interface{}) error {
	if dialogs.YesNoDialog("Would you like to enable camera interface?") {
		ssh, ok := storage["ssh"].(ssh_helper.Util)
		if !ok {
			return errors.New("Cannot get ssh config")
		}
		data := `
start_x=1
gpu_mem=128
`
		// disable_camera_led=1
		_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" >> %s`, data, bootMount+"config.txt"))
		if err != nil || strings.TrimSpace(eut) != "" {
			log.WithField("eut", eut).Error(err)
			return err
		}
	}
	return nil
}

// MountBoot is a method to attach image to loop and mount it
func (d *raspberryPi) MountBoot() error {
	log.Debug("Creating tmp folder")
	if err := d.exec(fmt.Sprintf("mkdir -p %s", bootMount)); err != nil {
		return err
	}

	log.Debug("Mounting boot partition")
	command := fmt.Sprintf("mount -o rw /dev/loop0p1 %s", bootMount)
	log.WithField("cmd", command).Debug("Mounting boot folder")
	if err := d.exec(command); err != nil {
		return err
	}
	return nil
}

// UnmountBoot is a method to unlink image folder and detach image from the loop
func (d *raspberryPi) UnmountBoot() error {
	log.Debug("Unlinking boot folder")
	command := fmt.Sprintf("umount %s", bootMount)
	if err := d.exec(command); err != nil {
		return err
	}

	log.Debug("Detaching image")
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
