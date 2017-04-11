package device

import (
	"fmt"

	"errors"

	"net"

	"strconv"

	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ssh_helper"
)

const (
	raspMount = "p2"
)

// raspberryPi device
type raspberryPi struct {
	*sdFlasher
}

// Configure overrides sdFlasher Configure() method with custom config
func (d *raspberryPi) Configure() error {
	job := help.NewBackgroundJob()
	c := config.NewDefault(d.conf.SSH)

	*(c.GetConfigFn(config.Interface)) = *config.NewConfigCallbackFn(configInterface, saveInterface)

	go func() {
		defer job.Close()

		if err := d.MountImg(raspMount); err != nil {
			job.Error(err)
		}
	}()

	// setup while background process mounting img
	if err := c.Setup(); err != nil {
		return err
	}

	if err := help.WaitJobAndSpin("waiting", job); err != nil {
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
		num := dialogs.SelectOneDialog("Please select a network interface:", device)
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
