package device

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/esp-flasher/common"
	"github.com/xshellinc/esp-flasher/esp"
	espFlasher "github.com/xshellinc/esp-flasher/esp/flasher"
	"github.com/xshellinc/esp-flasher/serialport"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/dialogs"
	"os/exec"
	"strings"
)

// serialFlasher is used for esp modules
type serialFlasher struct {
	*flasher
	port string
}

func (d *serialFlasher) Prepare() error {
	fmt.Println("[+] Enumerating serial ports...")
	port, err := serialport.GetPort("auto")
	if err != nil {
		return err
	}
	d.port = port
	fmt.Println("[+] Using ", dialogs.PrintColored(d.port))
	return nil
}

// Flash - override default flash
func (d *serialFlasher) Flash() error {

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
func (d *serialFlasher) Configure() error {
	c := config.New(d.conf.SSH)
	c.StoreValue("port", d.port)
	c.AddConfigFn(config.Wifi, config.NewCallbackFn(setWifi, saveWifi))

	if err := c.Setup(); err != nil {
		return err
	}

	if err := c.Write(); err != nil {
		return err
	}

	return nil
}

// Flash method is used to flash image to the sdcard
func (d *serialFlasher) Write() error {
	if !dialogs.YesNoDialog("Proceed to firmware flashing?") {
		log.Debug("Flash aborted")
		return nil
	}

	espFlashOpts := esp.FlashOpts{}
	espFlashOpts.ControlPort = d.port
	espFlashOpts.BaudRate = 460800
	espFlashOpts.BootFirmware = true
	espFlashOpts.MinimizeWrites = true

	fw, err := common.NewZipFirmwareBundle(d.devRepo.Image.URL)
	if err != nil {
		return err
	}

	log.Infof("Loaded %s/%s version %s (%s)\n", fw.Name, fw.Platform, fw.Version, fw.BuildID)

	switch strings.ToLower(fw.Platform) {
	case "esp32":
		err = espFlasher.Flash(esp.ChipESP32, fw, &espFlashOpts)
	case "esp8266":
		err = espFlasher.Flash(esp.ChipESP8266, fw, &espFlashOpts)
	default:
		err = fmt.Errorf("%s: unsupported platform '%s'", fw.Name, fw.Platform)
	}

	return err
}

func setWifi(storage map[string]interface{}) error {
	storage[config.Wifi+"_name"] = dialogs.GetSingleAnswer("WiFi SSID name: ", dialogs.EmptyStringValidator)
	storage[config.Wifi+"_pass"] = []byte(dialogs.WiFiPassword())
	return nil
}

// SaveWifi is a default method to save wpa_supplicant for the wifi connection
func saveWifi(storage map[string]interface{}) error {
	port := storage["port"].(string)

	if _, ok := storage[config.Wifi+"_name"]; !ok {
		return nil
	}

	command := fmt.Sprintf("stty -F %s 115200", port)
	if err := exec.Command(command).Run(); err != nil {
		return err
	}

	command = fmt.Sprintf("echo \"wifi set ssid %s\" > %s", port, storage[config.Wifi+"_name"])

	if err := exec.Command(command).Run(); err != nil {
		return err
	}

	command = fmt.Sprintf("echo \"wifi set password %s\" > %s", port, storage[config.Wifi+"_pass"])
	if err := exec.Command(command).Run(); err != nil {
		return err
	}

	return nil
}
