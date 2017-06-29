package main

import (
	"flag"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/esp-flasher/common"
	"github.com/xshellinc/esp-flasher/esp"
	espFlasher "github.com/xshellinc/esp-flasher/esp/flasher"
	"github.com/xshellinc/esp-flasher/serialport"
)

var (
	espFlashOpts esp.FlashOpts
	portFlag     *string
)

func init() {
	portFlag = flag.String("port", "auto", "Serial port where the device is connected. "+
		"If set to 'auto', ports on the system will be enumerated and the first will be used.")
	flag.UintVar(&espFlashOpts.BaudRate, "esp-baud-rate", 460800,
		"Data port speed during flashing")
	flag.StringVar(&espFlashOpts.DataPort, "esp-data-port", "",
		"If specified, this port will be used to send data during flashing. "+
			"If not set, --port is used.")
	flag.BoolVar(&espFlashOpts.InvertedControlLines, "esp-inverted-control-lines", false,
		"DTR and RTS control lines use inverted polarity")
	flag.StringVar(&espFlashOpts.FlashParams, "esp-flash-params", "",
		"Flash chip params. Either a comma-separated string of mode,size,freq or a number. "+
			"Mode must be one of: qio, qout, dio, dout. "+
			"Valid values for size are: 2m, 4m, 8m, 16m, 32m, 16m-c1, 32m-c1, 32m-c2. "+
			"If left empty, an attempt will be made to auto-detect. freq is SPI frequency "+
			"and can be one of 20m, 26m, 40m, 80m")
	flag.BoolVar(&espFlashOpts.EraseChip, "esp-erase-chip", false,
		"Erase entire chip before flashing")
	flag.BoolVar(&espFlashOpts.MinimizeWrites, "esp-minimize-writes", true,
		"Minimize the number of blocks to write by comparing current contents "+
			"with the images being written")
	flag.BoolVar(&espFlashOpts.BootFirmware, "esp-boot-after-flashing", true,
		"Boot the firmware after flashing")
	flag.StringVar(&espFlashOpts.ESP32EncryptionKeyFile, "esp32-encryption-key-file", "",
		"If specified, this file will be used to encrypt data before flashing. "+
			"Encryption is only applied to parts with encrypt=true.")
	flag.UintVar(&espFlashOpts.ESP32FlashCryptConf, "esp32-flash-crypt-conf", 0xf,
		"Value of the FLASH_CRYPT_CONF eFuse setting, affecting how key is tweaked.")
}

func main() {
	flag.Parse()

	fwname := "firmata32.zip"
	args := flag.Args()
	if len(args) == 2 {
		fwname = args[1]
	}
	fw, err := common.NewZipFirmwareBundle(fwname)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %s/%s version %s (%s)\n", fw.Name, fw.Platform, fw.Version, fw.BuildID)

	port, err := serialport.GetPort(*portFlag)
	if err != nil {
		log.Fatal(err)
	}

	switch strings.ToLower(fw.Platform) {
	case "esp32":
		espFlashOpts.ControlPort = port
		err = espFlasher.Flash(esp.ChipESP32, fw, &espFlashOpts)
	case "esp8266":
		espFlashOpts.ControlPort = port
		err = espFlasher.Flash(esp.ChipESP8266, fw, &espFlashOpts)
	default:
		err = fmt.Errorf("%s: unsupported platform '%s'", fwname, fw.Platform)
	}

	if err != nil {
		log.Fatal(err)
	}
	log.Println("Done")
}
