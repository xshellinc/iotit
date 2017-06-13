package device

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ssh_helper"
	"strings"
)

// serialFlasher is used for esp modules
type serialFlasher struct {
	*flasher
	port string
}

var flashsh = `flash`

// Configure method overrides generic flasher
func (d *serialFlasher) Configure() error {
	if err := d.Flash(); err != nil {
		return err
	}
	c := config.New(d.conf.SSH)
	c.StoreValue("port", d.port)
	c.AddConfigFn(config.NewCallbackFn(setWifi, saveWifi))

	if err := c.Setup(); err != nil {
		return err
	}
	if err := c.Write(); err != nil {
		return err
	}
	fmt.Println("[+] Done!")
	return d.Done()
}

// Flash method is used to flash image to the sdcard
func (d *serialFlasher) Flash() error {
	if !dialogs.YesNoDialog("Proceed to firmware burning?") {
		log.Debug("Flash aborted")
		return nil
	}

	fmt.Println("[+] Enumerating serial ports...")
	if err := d.getPort(); err != nil {
		return err
	}
	fmt.Println("[+] Using ", dialogs.PrintColored(d.port))
	args := []string{
		fmt.Sprintf("%s@%s", vbox.VBoxUser, vbox.VBoxIP),
		"-p",
		vbox.VBoxSSHPort,
		"flash32.sh",
		d.port,
		"/tmp/fw/bootloader.bin",
		"/tmp/fw/isaax_firmata.bin",
		"/tmp/fw/partitions_singleapp.bin",
	}
	if err := help.ExecStandardStd("ssh", args...); err != nil {
		return err
	}

	return nil
}

func (d *serialFlasher) getPort() error {
	list := d.enumerateSerialPorts()
	if len(list) == 0 {
		return errors.New("No ports found")
	}
	// we are inside VM, so we assume it's unlikely there will be more then one port
	// and even if there are more, there is no way for us to tell user what is what.
	d.port = list[0]
	return nil
}

func (d *serialFlasher) enumerateSerialPorts() []string {
	out := ""
	list := []string{}
	if err := d.execOverSSH("ls -1 /dev/ttyUSB*", &out); err != nil {
		log.Error(err)
		return list
	}
	list = strings.Split(out, "\n")
	log.WithField("out", out).WithField("list", list).Info("ports")
	return list
}

// Done prints out final success message
func (d *serialFlasher) Done() error {
	if err := vbox.Stop(d.vbox.UUID); err != nil {
		log.Error(err)
	}

	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t Module flashed and configured")
	fmt.Println(strings.Repeat("*", 100))

	return nil
}

func setWifi(storage map[string]interface{}) error {
	storage[config.GetConstLiteral(config.Wifi)+"_name"] = dialogs.GetSingleAnswer("WiFi SSID name: ", dialogs.EmptyStringValidator)
	storage[config.GetConstLiteral(config.Wifi)+"_pass"] = []byte(dialogs.WiFiPassword())
	return nil
}

// SaveWifi is a default method to save wpa_supplicant for the wifi connection
func saveWifi(storage map[string]interface{}) error {
	if _, ok := storage[config.GetConstLiteral(config.Wifi)+"_name"]; !ok {
		return nil
	}
	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}
	port := storage["port"]
	command := fmt.Sprintf("stty -F %s 115200", port)
	if _, eut, err := ssh.Run(command); err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	command = fmt.Sprintf("echo \"wifi set ssid %s\" > %s", port, storage[config.GetConstLiteral(config.Wifi)+"_name"])

	if _, eut, err := ssh.Run(command); err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	command = fmt.Sprintf("echo \"wifi set password %s\" > %s", port, storage[config.GetConstLiteral(config.Wifi)+"_pass"])
	if _, eut, err := ssh.Run(command); err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

func (d *serialFlasher) execOverSSH(command string, outp *string) error {
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error executing: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" {
		log.Debug(strings.TrimSpace(out))
		if outp != nil {
			*outp = strings.TrimSpace(out)
		}
	}
	return nil
}
