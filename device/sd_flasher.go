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

	if d.img == "" {
		return errors.New("Image not found, please check if the repo is valid")
	}

	command := fmt.Sprintf("losetup -f -P %s", help.AddPathSuffix("unix", constants.TMP_DIR, d.img))
	if err := d.execOverSSH(command, nil); err != nil {
		return err
	}

	log.Debug("Creating tmp folder")
	command = fmt.Sprintf("mkdir -p %s", constants.MountDir)
	if err := d.execOverSSH(command, nil); err != nil {
		return err
	}
	if loopMount == "" {
		command = fmt.Sprintf("ls /dev/loop0p*")
		compiler, _ := regexp.Compile(`loop0p[\d]+`)

		out := ""
		if err := d.execOverSSH(command, &out); err != nil {
			return err
		}
		opts := compiler.FindAllString(out, -1)
		if len(opts) == 0 {
			return errors.New("Cannot find a mounting point")
		}
		unmount := fmt.Sprintf("umount %s", constants.MountDir)
		for _, loop := range opts {
			command = fmt.Sprintf("%s -o rw /dev/%s %s", constants.Mount, loop, constants.MountDir)
			if err := d.execOverSSH(command, nil); err != nil {
				return err
			}
			command = fmt.Sprintf("ls %s", constants.MountDir)
			out := ""
			if err := d.execOverSSH(command, &out); err != nil {
				return err
			}
			if !strings.Contains(out, "etc") && !strings.Contains(out, "opt") {
				if err := d.execOverSSH(unmount, nil); err != nil {
					return err
				}
			} else {
				return nil
			}
		}
		return errors.New("Can't find linux root partition inside that image")
	}
	log.Debug("Mounting sd folder on", loopMount)
	command = fmt.Sprintf("%s -o rw /dev/loop0%s %s", constants.Mount, loopMount, constants.MountDir)
	if err := d.execOverSSH(command, nil); err != nil {
		return err
	}
	return nil
}

// UnmountImg is a method to unlink image folder and detach image from the loop
func (d *sdFlasher) UnmountImg() error {
	log.Debug("Unmounting image folder")
	command := fmt.Sprintf("umount %s", constants.MountDir)
	if err := d.execOverSSH(command, nil); err != nil {
		return err
	}

	log.Debug("Detaching image loop device")
	command = "losetup -D"
	if err := d.execOverSSH(command, nil); err != nil {
		return err
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

	log.Debug("Downloading image from vbox")

	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		if err := d.conf.SSH.ScpFrom(help.AddPathSuffix("unix", constants.TMP_DIR, d.img), filepath.Join(help.GetTempDir(), d.img)); err != nil {

			job.Error(err)
		}
	}()
	if err := help.WaitJobAndSpin("Copying files", job); err != nil {
		log.Error(err)
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
	//fmt.Println(strings.Repeat("*", 100))
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

	fmt.Printf("\n\t\t Flashing Complete!\n")
	fmt.Printf("\t\t Please insert your sd card into your %s\n", d.device)
	fmt.Println("\t\t ssh to your board with the following credentials")
	fmt.Printf("\t\t ssh username:"+dialogs.PrintColored("%s")+" password:"+dialogs.PrintColored("%s")+"\n",
		d.devRepo.Image.User, d.devRepo.Image.Pass)
	fmt.Println("\t\t If you have any question or suggestions feel free to make an issue at https://github.com/xshellinc/iotit/issues/ or tweet us @isaax_iot")

	return nil
}

func (d *sdFlasher) execOverSSH(command string, outp *string) error {
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
