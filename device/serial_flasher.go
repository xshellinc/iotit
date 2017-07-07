package device

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/esp-flasher/common"
	"github.com/xshellinc/esp-flasher/esp"
	espFlasher "github.com/xshellinc/esp-flasher/esp/flasher"
	"github.com/xshellinc/esp-flasher/serialport"
	"github.com/xshellinc/go-serial"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/ssh_helper"
	"strings"
	"time"
)

// serialFlasher is used for esp modules
type serialFlasher struct {
	*flasher
	Port string
}

func (d *serialFlasher) Prepare() error {
	log.Debug("Prepare")
	if len(d.Port) == 0 {
		fmt.Println("[+] Enumerating serial ports...")
		port, err := serialport.GetPort("auto")
		if err != nil {
			return err
		}
		d.Port = port
	}
	fmt.Println("[+] Using ", dialogs.PrintColored(d.Port))
	return nil
}

// Flash - override default flash
func (d *serialFlasher) Flash() error {
	log.Debug("Serial flasher")

	if err := d.Prepare(); err != nil {
		return err
	}

	if err := d.Write(); err != nil {
		return err
	}
	time.Sleep(time.Second * 3) // wait for the module to boot
	if err := d.Configure(); err != nil {
		return err
	}

	return d.Done()
}

// Configure method overrides generic flasher
func (d *serialFlasher) Configure() error {
	log.WithField("device", "serial").Debug("Configure")
	fmt.Println("[+] Configuring...")

	if len(d.Port) == 0 {
		d.Prepare()
	}

	if !d.Quiet {
		c := config.New(ssh_helper.New("", "", "", ""))
		commonOpts := serial.OpenOptions{
			BaudRate:              115200,
			DataBits:              8,
			ParityMode:            serial.PARITY_NONE,
			StopBits:              1,
			InterCharacterTimeout: 200.0,
			PortName:              d.Port,
		}
		sc, err := serial.Open(commonOpts)
		if err != nil {
			return err
		}
		defer sc.Close()

		c.StoreValue("port", sc)
		c.AddConfigFn(config.Wifi, config.NewCallbackFn(setWifi, saveWifi))

		if err := c.Setup(); err != nil {
			return err
		}

		if err := c.Write(); err != nil {
			return err
		}
		fmt.Println("[+] Module configured")
	}

	return nil
}

// Flash method is used to flash image to the sdcard
func (d *serialFlasher) Write() error {
	if !d.Quiet {
		if !dialogs.YesNoDialog("Proceed to firmware flashing?") {
			log.Debug("Flash aborted")
			return nil
		}
	}

	espFlashOpts := esp.FlashOpts{}
	espFlashOpts.ControlPort = d.Port
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

// Done prints out final success message
func (d *serialFlasher) Done() error {
	fmt.Println("\t\t ...                      .................    ..                ")
	fmt.Println("\t\t ...                      .................   ....    ...        ")
	fmt.Println("\t\t ...                             ....                 ...        ")
	fmt.Println("\t\t ...          .....              ....                 ...        ")
	fmt.Println("\t\t ...       ...........           ....         ...     .......... ")
	fmt.Println("\t\t ...      ...       ...          ....         ...     ...        ")
	fmt.Println("\t\t ...     ...         ...         ....         ...     ...        ")
	fmt.Println("\t\t ...     ...         ...         ....         ...     ...        ")
	fmt.Println("\t\t ...     ...         ...         ....         ...     ...        ")
	fmt.Println("\t\t ...     ....       ....         ....         ...      ...       ")
	fmt.Println("\t\t ...      .....   .....          ....         ...      ....   .. ")
	fmt.Println("\t\t ...         .......             ....         ...        ....... ")
	fmt.Println("\n\t\t Flashing Complete!")
	fmt.Println("\t\t If you have any questions or suggestions feel free to make an issue at https://github.com/xshellinc/iotit/issues/ or tweet us @isaax_iot")
	return nil
}

func setWifi(storage map[string]interface{}) error {
	storage[config.Wifi+"_name"] = dialogs.GetSingleAnswer("WiFi SSID name: ", dialogs.EmptyStringValidator)
	storage[config.Wifi+"_pass"] = []byte(dialogs.WiFiPassword())
	return nil
}

// SaveWifi is a default method to save wpa_supplicant for the wifi connection
func saveWifi(storage map[string]interface{}) error {
	port := storage["port"].(serial.Serial)

	if _, ok := storage[config.Wifi+"_name"]; !ok {
		return nil
	}

	if n, err := port.Write([]byte(fmt.Sprintf("wifi set ssid %s\r\n", storage[config.Wifi+"_name"]))); err != nil {
		return err
	} else {
		log.Debug("Written:", n)
	}
	data := make([]byte, 10000)
	if _, err := port.Read(data); err != nil {
		return err
	}
	//  else {
	// 	log.WithField("data", string(data)).Debug("Response")
	// }
	if n, err := port.Write([]byte(fmt.Sprintf("wifi set password %s\r\n", storage[config.Wifi+"_pass"]))); err != nil {
		return err
	} else {
		log.Debug("Written:", n)
	}
	data = make([]byte, 10000)
	if _, err := port.Read(data); err != nil {
		return err
	}
	if n, err := port.Write([]byte("wifi start\r\n")); err != nil {
		return err
	} else {
		log.Debug("Written:", n)
	}
	data = make([]byte, 10000)
	if _, err := port.Read(data); err != nil {
		return err
	}
	return nil
}
