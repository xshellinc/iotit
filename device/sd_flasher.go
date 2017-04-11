package device

import (
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/device/workstation"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

// SdFlasher interface defines all methods for sdFlasher
type SdFlasher interface {
	MountImg() error
	UnmountImg() error
	Flash() error
	Done() error
}

// sdFlasher is a used as a generic flasher for devices except raspberrypi/nanopi and others defined in the device package
type sdFlasher struct {
	*flasher
}

// MountImg is a method to attach image to loop and mount it
func (d *sdFlasher) MountImg(loopMount string) error {
	if loopMount == "" {
		return errors.New("Application error: Nothing to mount")
	}

	logrus.Debug("Attaching an image")
	command := fmt.Sprintf("losetup -f -P %s", filepath.Join(constants.TMP_DIR, d.img))
	out, eut, err := d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out, eut)

	logrus.Debug("Creating tmp folder")
	command = fmt.Sprintf("mkdir -p %s", constants.MountDir)
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out, eut)

	logrus.Debug("Mounting tmp folder")
	command = fmt.Sprintf("%s -o rw /dev/loop0%s %s", constants.Mount, loopMount, constants.MountDir)
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out, eut)

	return nil
}

// UnmountImg is a method to unlink image folder and detach image from the loop
func (d *sdFlasher) UnmountImg() error {
	logrus.Debug("Unlinking image folder")
	command := fmt.Sprintf("umount %s", constants.MountDir)
	out, eut, err := d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out)

	logrus.Debug("Detaching and image")
	command = "losetup -D"
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out)

	return nil
}

// Flash method is used to flash image into the sdcard
func (d *sdFlasher) Flash() error {
	logrus.Debug("Downloading an image from vbox")

	logrus.Debug("Copying files from vbox")
	fmt.Println("[+] Copying files...")
	err := d.conf.SSH.ScpFromServer(help.AddPathSuffix("unix", constants.TMP_DIR, d.img), filepath.Join(constants.TMP_DIR, d.img))
	if err != nil {
		return err
	}

	w := workstation.NewWorkStation()
	img := filepath.Join(constants.TMP_DIR, d.img)

	logrus.Debug("Writing the image into sd card")
	job, err := w.WriteToDisk(img)
	if err != nil {
		return err
	}
	if job != nil {
		if err := help.WaitJobAndSpin("flashing", job); err != nil {
			return err
		}
	}

	logrus.Debug("Removing sd from dir")
	if err = os.Remove(img); err != nil {
		logrus.Error("[-] Can not remove image: " + err.Error())
	}

	if err = w.Unmount(); err != nil {
		logrus.Error("Error parsing mount option ", "error msg:", err.Error())
	}
	if err = w.Eject(); err != nil {
		logrus.Error("Error parsing mount option ", "error msg:", err.Error())
	}

	if err = vbox.Stop(d.vbox.UUID); err != nil {
		logrus.Error(err)
	}

	return nil
}

// Configure method overrides generic flasher method and includes logic of mounting configuring and flashing the device into the sdCard
func (d *sdFlasher) Configure() error {

	c := config.NewDefault(d.conf.SSH)

	if err := d.MountImg(""); err != nil {
		return err
	}

	// setup while background process mounting img
	if err := c.Setup(); err != nil {
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

// Done prints out final success message
func (d *sdFlasher) Done() error {
	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t SD CARD READY!  \t\t\t\t\t\t\t\t   *")
	fmt.Printf("*\t\t PLEASE INSERT YOUR SD CARD TO YOUR %s \t\t\t\t\t   *\n", d.device)
	fmt.Println("*\t\t IF YOU HAVE NOT SET UP THE USB WIFI, PLEASE CONNECT TO ETHERNET \t\t   *")
	fmt.Printf("*\t\t SSH USERNAME:\x1b[31m%s\x1b[0m PASSWORD:\x1b[31m%s\x1b[0m \t\t\t\t\t\t\t   *\n",
		d.devRepo.Image.User, d.devRepo.Image.Pass)
	fmt.Println(strings.Repeat("*", 100))

	return nil
}
