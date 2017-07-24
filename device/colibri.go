package device

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/esp-flasher/serialport"
	"github.com/xshellinc/go-serial"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/workstation"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

const (
	COLIBRI = "colibri"
)

var serialPort serial.Serial

// toradex colibri imx6 device
type colibri struct {
	*flasher
	Port string
}

// Prepare overrides flasher Prepare method with port initialization
func (d *colibri) Prepare() error {
	// start VM, upload image and extract it
	d.flasher.Prepare()
	log.WithField("device", COLIBRI).Debug("Prepare")
	// install toradex flasher dependencies
	d.installTools()

	fmt.Println("[+] Enumerating serial ports...")
	port, err := serialport.GetPort("auto")
	if err != nil {
		return err
	}
	d.Port = port

	fmt.Println("[+] Using ", dialogs.PrintColored(d.Port))
	return nil
}

// Configure overrides flasher Configure() method with custom image configuration
func (d *colibri) Configure() error {
	log.WithField("device", COLIBRI).Debug("Configure")
	fmt.Println("[+] Configuring...")

	job := help.NewBackgroundJob()

	go func() {
		defer job.Close()

		if err := d.configureImage(); err != nil {
			job.Error(err)
			return
		}
	}()

	if err := help.WaitJobAndSpin("Waiting", job); err != nil {
		return err
	}

	fmt.Println("[+] Image ready")
	return nil
}

// Flash configures and flashes image
func (d *colibri) Flash() error {
	if err := d.Prepare(); err != nil {
		return err
	}

	if err := d.Configure(); err != nil {
		return err
	}

	commonOpts := serial.OpenOptions{
		BaudRate:              115200,
		DataBits:              8,
		ParityMode:            serial.PARITY_NONE,
		StopBits:              1,
		InterCharacterTimeout: 200.0,
		PortName:              d.Port,
	}
	var err error
	serialPort, err = serial.Open(commonOpts)
	if err != nil {
		return err
	}

	defer serialPort.Close()

	if err := d.Write(); err != nil {
		return err
	}

	if err := d.runUpdate(); err != nil {
		return err
	}

	return d.Done()
}

func (d *colibri) installTools() error {
	if err := d.exec("apk add dosfstools parted sudo e2fsprogs-extra coreutils"); err != nil {
		return err
	}
	return nil
}

func (d *colibri) configureImage() error {
	d.img = "colibri_image.tar"

	log.Debug("Creating tmp folder")
	command := fmt.Sprintf("mkdir -p %s", config.MountDir)
	if err := d.exec(command); err != nil {
		return err
	}
	log.Debug("Running update.sh")
	command = fmt.Sprintf("cd %s && ./update.sh -o %s", help.AddPathSuffix("unix", config.TmpDir, d.folder), config.MountDir)
	if err := d.exec(command); err != nil {
		return err
	}
	command = fmt.Sprintf("tar -C %s -cf %s .", config.MountDir, help.AddPathSuffix("unix", config.TmpDir, "colibri_image.tar"))
	if err := d.exec(command); err != nil {
		return err
	}
	return nil
}

// Write - writes image to SD card
func (d *colibri) Write() error {
	log.WithField("img", d.img).Debug("Downloading image from vbox")

	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		if err := d.conf.SSH.ScpFrom(help.AddPathSuffix("unix", config.TmpDir, d.img), filepath.Join(help.GetTempDir(), d.img)); err != nil {
			job.Error(err)
		}
	}()

	if err := help.WaitJobAndSpin("Copying files", job); err != nil {
		log.Error(err)
		return err
	}

	fmt.Println("[+] Listing available disks...")
	w := workstation.NewWorkStation("")
	img := filepath.Join(help.GetTempDir(), d.img)

	log.WithField("img", img).Debug("Writing image to disk")

	if job, err := w.CopyToDisk(img); err != nil {
		return err
	} else if job != nil {
		if err := help.WaitJobAndSpin("Flashing", job); err != nil {
			return err
		}
	}
	time.Sleep(time.Second * 2)
	w.Unmount()
	fmt.Println("[+] SD card prepared")
	return nil
}

func (d *colibri) runUpdate() error {
	for !dialogs.YesNoDialog("Please insert prepared SD card into your Colibri iMX6 board. Type yes once ready.") {
	}

	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		for {
			if _, err := serialPort.Write([]byte(" ")); err != nil {
				job.Error(err)
			}
			data := make([]byte, 10000)
			if _, err := serialPort.Read(data); err == nil {
				log.WithField("data", string(data)).Debug("Response")
				if strings.Contains(string(data), "Colibri iMX6 #") || strings.Contains(string(data), "iMX6 #") {
					break
				}
			}
		}
	}()

	if err := help.WaitJobAndSpin("Now reset or power up the board", job); err != nil {
		log.Error(err)
		return err
	}
	fmt.Println("[+] Flashing to eMMC")
	if _, err := serialPort.Write([]byte("run setupdate\r\n")); err != nil {
		log.Error(err)
	}
	data := make([]byte, 10000)
	if _, err := serialPort.Read(data); err != nil {
		log.Error(err)
	} else {
		log.WithField("data", string(data)).Debug("Response")
	}
	time.Sleep(time.Second * 1)
	data = make([]byte, 10000)
	if _, err := serialPort.Read(data); err != nil {
		log.Error(err)
	} else {
		log.WithField("data", string(data)).Debug("Response")
	}

	time.Sleep(time.Second * 2)
	if n, err := serialPort.Write([]byte("run update\r\n")); err != nil {
		log.Error(err)
	} else {
		log.Debug("Written:", n)
	}
	data = make([]byte, 10000)
	if _, err := serialPort.Read(data); err != nil {
		log.Error(err)
	} else {
		log.WithField("data", string(data)).Debug("Response")
	}
	for {
		serialPort.SetReadTimeout(time.Second * 5)
		data = make([]byte, 1024)
		if _, err := serialPort.Read(data); err != nil {
			log.Error(err)
		} else {
			line := strings.TrimSpace(string(data))
			log.WithField("data", line).Debug("Response")
			fmt.Print(line)
			if strings.Contains(line, "resetting") {
				fmt.Println("[+] Done! Rebooting the board.")
				return nil
			}
		}
	}

	return nil
}

func (d *colibri) exec(command string) error {
	log.Debug(command)
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error executing: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" {
		log.Debug(strings.TrimSpace(out))
	}
	return nil
}
