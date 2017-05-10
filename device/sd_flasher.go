package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/device/workstation"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
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
	log.Debug("Attaching an image")
	command := fmt.Sprintf("losetup -f -P %s", help.AddPathSuffix("unix", constants.TMP_DIR, d.img))
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error when execute: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" || strings.TrimSpace(eut) != "" {
		log.Debug(out, eut)
	}

	log.Debug("Creating tmp folder")
	command = fmt.Sprintf("mkdir -p %s", constants.MountDir)
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error when execute: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" || strings.TrimSpace(eut) != "" {
		log.Debug(out, eut)
	}

	if loopMount == "" {
		command = fmt.Sprintf("ls /dev/loop0p*")
		compiler, _ := regexp.Compile(`loop0p[\d]+`)
		if out, eut, err := d.conf.SSH.Run(command); err != nil {
			log.Error("[-] Error when execute: ", command, eut)
			return err
		} else {
			log.Debug(out, eut)
			opts := compiler.FindAllString(out, -1)
			if len(opts) == 0 {
				return errors.New("Cannot find a mounting point")
			}
			loopMount = opts[dialogs.SelectOneDialog("Please select a correct mounting point: ", opts)]
			loopMount = loopMount[5:]
		}
	}

	log.Debug("Mounting tmp folder")
	command = fmt.Sprintf("%s -o rw /dev/loop0%s %s", constants.Mount, loopMount, constants.MountDir)
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error when execute: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" || strings.TrimSpace(eut) != "" {
		log.Debug(out, eut)
	}

	return nil
}

// UnmountImg is a method to unlink image folder and detach image from the loop
func (d *sdFlasher) UnmountImg() error {
	log.Debug("Unmounting image folder")
	command := fmt.Sprintf("umount %s", constants.MountDir)
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error when execute: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" {
		log.Debug(strings.TrimSpace(out))
	}

	log.Debug("Detaching image loop device")
	command = "losetup -D"
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error when execute: ", command, eut)
		return err
	} else if strings.TrimSpace(out) != "" {
		log.Debug(strings.TrimSpace(out))
	}
	return nil
}

// Flash method is used to flash image to the sdcard
func (d *sdFlasher) Flash() error {
	if !dialogs.YesNoDialog("Proceed to image burning?") {
		log.Debug("Aborted")
		return nil
	}

	help.DeleteFile(filepath.Join(help.GetTempDir(), d.img))

	fmt.Println("[+] Copying files...")
	err := d.conf.SSH.ScpFromServer(help.AddPathSuffix("unix", constants.TMP_DIR, d.img),
		filepath.Join(help.GetTempDir(), d.img))
	if err != nil {
		return err
	}

	fmt.Println("[+] Listing available disks...")
	w := workstation.NewWorkStation()
	img := filepath.Join(help.GetTempDir(), d.img)

	log.WithField("img", img).Debug("Writing image to disk")
	if job, err := w.WriteToDisk(img); err != nil {
		return err
	} else if job != nil {
		if err := help.WaitJobAndSpin("Flashing", job); err != nil {
			return err
		}
	}

	log.Debug("Removing sd from dir")
	if err := os.Remove(img); err != nil {
		log.Error("Can not remove image: " + err.Error())
	}

	if err := w.Unmount(); err != nil {
		log.Error("Error parsing mount option ", "error msg:", err.Error())
	}
	if err := w.Eject(); err != nil {
		log.Error("Error parsing mount option ", "error msg:", err.Error())
	}

	if err := vbox.Stop(d.vbox.UUID); err != nil {
		log.Error(err)
	}

	return nil
}

// Configure method overrides generic flasher method and includes logic of mounting configuring and flashing the device into the sdCard
func (d *sdFlasher) Configure() error {

	c := config.NewDefault(d.conf.SSH)

	if err := d.MountImg(fmt.Sprintf("")); err != nil {
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
	fmt.Println("*\t\t SD CARD READY!")
	fmt.Printf("*\t\t PLEASE INSERT YOUR SD CARD TO YOUR %s\n", d.device)
	fmt.Println("*\t\t IF YOU HAVE NOT SET UP THE USB WIFI, PLEASE CONNECT TO ETHERNET")
	fmt.Printf("*\t\t SSH USERNAME:"+dialogs.PrintColored("%s")+" PASSWORD:"+dialogs.PrintColored("%s")+"\n",
		d.devRepo.Image.User, d.devRepo.Image.Pass)
	fmt.Println(strings.Repeat("*", 100))

	return nil
}
