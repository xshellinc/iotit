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

var serialPort serial.Serial

const portSelectionTries = 3

// toradex colibri imx6 device
type colibri struct {
	*flasher
	Port string
	Disk string
}

// Prepare overrides flasher Prepare method with port initialization
func (d *colibri) Prepare() error {
	// start VM, upload image and extract it
	d.flasher.Prepare()
	log.WithField("device", "colibri").Debug("Prepare")
	// install toradex flasher dependencies
	d.installTools()
	return nil
}

func (d *colibri) getPort() error {
	if d.Port != "" {
		return nil
	}
	var perr error
	for attempt := 0; attempt < portSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("No ports found. Reconnect your device and try again. Ready?") {
			return perr
		}
		fmt.Println("[+] Enumerating serial ports...")
		port, err := serialport.GetPort("auto")
		if err != nil {
			perr = err
			continue
		}
		d.Port = port
		break
	}
	if d.Port == "" {
		return perr
	}
	fmt.Println("[+] Using ", dialogs.PrintColored(d.Port))
	return nil
}

// Configure overrides flasher Configure() method with custom image configuration
func (d *colibri) Configure() error {
	log.WithField("device", "colibri").Debug("Configure")
	fmt.Println("[+] Configuring...")

	job := help.NewBackgroundJob()

	go func() {
		defer job.Close()

		if err := d.configureImage(); err != nil {
			job.Error(err)
			return
		}
	}()

	if err := help.WaitJobAndSpin("Preparing", job); err != nil {
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

	if err := d.Write(); err != nil {
		return err
	}

	return d.Done()
}

func (d *colibri) installTools() error {
	if err := d.exec("apk add dosfstools parted sudo e2fsprogs-extra coreutils libattr zip"); err != nil {
		return err
	}
	return nil
}

func (d *colibri) configureImage() error {
	d.img = "colibri_image.zip"

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
	command = fmt.Sprintf("cd %s && zip -0 -r ../%s *", config.MountDir, d.img)
	if err := d.exec(command); err != nil {
		return err
	}
	return nil
}

// Write - writes image to SD card
func (d *colibri) Write() error {
	if d.img != "" {
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
	} else {
		d.img = "colibri_image.zip"
	}

	flashOnly := d.flasher.CLI.Bool("flash")
	if !flashOnly {
		w := workstation.NewWorkStation(d.Disk)
		img := filepath.Join(help.GetTempDir(), d.img)

		log.WithField("img", img).Debug("Writing image to disk")

		if job, err := w.CopyToDisk(img); err != nil {
			return err
		} else if job != nil {
			if err := help.WaitJobAndSpin("Writing image files to SD card", job); err != nil {
				return err
			}
		}
		time.Sleep(time.Second * 2)
		w.Unmount()
		fmt.Println("[+] SD card prepared")
	}
	if d.Port == "" {
		if err := d.getPort(); err != nil {
			log.Error(err)
			return err
		}
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

	if err := d.runUpdate(); err != nil {
		return err
	}

	return nil
}

func readResponse() string {
	data := make([]byte, 10000)
	if _, err := serialPort.Read(data); err != nil {
		log.Error(err)
	}
	line := strings.TrimSpace(string(data))
	log.WithField("data", line).Debug("Response")
	return line
}

func rebootBoard() error {
	serialPort.SetReadTimeout(time.Second * 5)
	if _, err := serialPort.Write([]byte("\r\n")); err != nil {
		log.Error(err)
		serialPort.Write([]byte("\r\n"))
	}
	for {
		line := readResponse()
		if line != "" {
			if strings.Contains(line, "imx6 login:") {
				if _, err := serialPort.Write([]byte("root\r\n")); err != nil {
					log.Error(err)
				}
				readResponse() //command echo
				readResponse() //response
				time.Sleep(time.Second * 2)
				if _, err := serialPort.Write([]byte("reboot\r\n")); err != nil {
					log.Error(err)
				}
				readResponse() //command echo
				readResponse() //response
				time.Sleep(time.Second * 2)
				return nil
			}
		}
	}
}

func bootInRecovery() *help.BackgroundJob {
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
	return job
}

func (d *colibri) runUpdate() error {
	for !dialogs.YesNoDialog("Please insert prepared SD card into your Colibri iMX6 board. Type yes once ready.") {
	}
	message := "Now reset or power up the board"
	if !dialogs.YesNoDialog("Do you have a reset button on your carrier board?") {
		fmt.Println("[+] Trying to reboot the board")
		rebootBoard()
		message = "Booting in recovery"
	}
	job := bootInRecovery()
	if err := help.WaitJobAndSpin(message, job); err != nil {
		log.Error(err)
		return err
	}
	fmt.Println("[+] Flashing to eMMC")
	if _, err := serialPort.Write([]byte("run setupdate\r\n")); err != nil {
		log.Error(err)
	}

	readResponse()
	time.Sleep(time.Second * 1)
	readResponse()
	time.Sleep(time.Second * 2)
	if n, err := serialPort.Write([]byte("run update\r\n")); err != nil {
		log.Error(err)
	} else {
		log.Debug("Written:", n)
	}
	readResponse()
	for {
		serialPort.SetReadTimeout(time.Second * 5)
		line := readResponse()
		fmt.Print(line)
		if strings.Contains(line, "resetting") {
			fmt.Println("[+] Done! Rebooting the board.")
			return nil
		}
	}
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
