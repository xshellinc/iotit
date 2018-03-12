package device

import (
	"fmt"
	"path/filepath"
	"strings"

	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/workstation"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// sdFlasher is a used as a generic flasher for devices except raspberrypi/nanopi and others defined in the device package
type sdFlasher struct {
	*flasher
	Disk string
}

// MountImg is a method to attach image to loop and mount it
func (d *sdFlasher) MountImg(loopMount string) error {
	log.WithField("img", d.img).Debug("Attaching an image")

	if d.img == "" {
		return fmt.Errorf("Image not found, please check if the repo is valid")
	}

	command := fmt.Sprintf("losetup -f -P %s", help.AddPathSuffix("unix", config.TmpDir, d.img))
	if err := d.execOverSSH(command, nil); err != nil {
		return err
	}

	log.Debug("Creating tmp folder")
	command = fmt.Sprintf("mkdir -p %s", config.MountDir)
	if err := d.execOverSSH(command, nil); err != nil {
		return err
	}
	d.mounted = false
	if loopMount == "" {
		log.Debug("Empty loopMount, trying to detect linux partition")
		command = fmt.Sprintf("ls /dev/loop0p*")
		compiler, _ := regexp.Compile(`loop0p[\d]+`)

		out := ""
		if err := d.execOverSSH(command, &out); err != nil {
			return err
		}
		opts := compiler.FindAllString(out, -1)
		if len(opts) == 0 {
			log.Info("Cannot find a mounting point")
			return nil
		}
		unmount := fmt.Sprintf("umount %s", config.MountDir)
		for _, loop := range opts {
			log.WithField("loop", loop).Debug("Iterating over partitions")
			command = fmt.Sprintf("mount -o rw /dev/%s %s", loop, config.MountDir)
			if err := d.execOverSSH(command, nil); err != nil {
				if strings.Contains(err.Error(), "wrong fs type, bad option, bad superblock") {
					if err := d.execOverSSH("dmesg|tail -n 1", &out); err != nil {
						log.Error(err)
						continue
					}
					log.Debug(out)

					compiler2, _ := regexp.Compile(`block count (\d+) exceeds size of device \((\d+) blocks\)`)
					result := compiler2.FindAllSubmatch([]byte(out), -1)
					if len(result) == 0 {
						log.Error("block info not found")
						continue
					}
					if err := d.execOverSSH("apk add e2fsprogs-extra", nil); err != nil {
						log.Error(err)
						continue
					}
					command = fmt.Sprintf("resize2fs -f /dev/%s %s", loop, string(result[0][2]))
					d.execOverSSH(command, nil)
					command = fmt.Sprintf("mount -o rw /dev/%s %s", loop, config.MountDir)
					if err := d.execOverSSH(command, nil); err != nil {
						log.Error(err)
						continue
					}
				} else {
					log.WithField("type", "not bad superblock").Error(err)
					continue
				}
			}
			command = fmt.Sprintf("ls %s", config.MountDir)
			out := ""
			if err := d.execOverSSH(command, &out); err != nil {
				log.Error(err)
				continue
			}
			if !strings.Contains(out, "etc") && !strings.Contains(out, "opt") {
				if err := d.execOverSSH(unmount, nil); err != nil {
					log.Error(err)
				}
				continue
			}
			d.mounted = true
			return nil
		}
		log.Info("Can't find linux root partition inside that image")
		return nil
	}
	log.Debug("Mounting sd folder on", loopMount)
	command = fmt.Sprintf("mount -o rw /dev/loop0%s %s", loopMount, config.MountDir)
	if err := d.execOverSSH(command, nil); err != nil {
		return err
	}
	d.mounted = true
	return nil
}

// UnmountImg is a method to unlink image folder and detach image from the loop
func (d *sdFlasher) UnmountImg() error {
	if d.mounted {
		return nil
	}
	log.Debug("Unmounting image folder")
	command := fmt.Sprintf("umount %s", config.MountDir)
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
func (d *sdFlasher) Write() error {
	if !d.Quiet {
		if !dialogs.YesNoDialog("Proceed to image flashing?") {
			log.Debug("Aborted")
			return nil
		}
	}

	help.DeleteFile(filepath.Join(help.GetTempDir(), d.img))

	log.Debug("Downloading image from vbox")

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

	w := workstation.NewWorkStation(d.Disk)
	img := filepath.Join(help.GetTempDir(), d.img)

	log.WithField("img", img).Debug("Writing image to disk")
	if job, err := w.WriteToDisk(img); err != nil {
		return err
	} else if job != nil {
		if err := help.WaitJobAndSpin("Flashing", job); err != nil {
			return err
		}
	}

	if err := w.Unmount(); err != nil {
		log.Error("Error parsing mount option ", "error msg:", err.Error())
	}
	if err := w.Eject(); err != nil {
		log.Error("Error parsing mount option ", "error msg:", err.Error())
	}

	if err := d.conf.Stop(d.Quiet); err != nil {
		log.Error(err)
	}

	return d.Done()
}

// Configure method overrides generic flasher method and includes logic of mounting configuring and flashing the device into the sdCard
func (d *sdFlasher) Configure() error {
	if err := d.Prepare(); err != nil {
		return err
	}

	log.WithField("device", "SD").Debug("Configure")
	c := config.NewDefault(d.conf.SSH)

	if err := d.MountImg(""); err != nil {
		return err
	}
	if !d.mounted {
		if !dialogs.YesNoDialog("IoTit can't configure this image because no linux partitions were found inside. Do you want to proceed to image writing anyway?") {
			return fmt.Errorf("Aborted")
		}
		return nil
	}
	if !d.Quiet {
		if dialogs.YesNoDialog("Would you like to configure your board?") {
			if err := c.Setup(); err != nil {
				return err
			}

			// write configs that were setup above
			if err := c.Write(); err != nil {
				return err
			}
		}
	}

	if err := d.UnmountImg(); err != nil {
		return err
	}

	return nil
}

// Flash configures and flashes image
func (d *sdFlasher) Flash() error {
	log.Debug("SD flasher")

	if err := d.Configure(); err != nil {
		return err
	}

	if err := d.Write(); err != nil {
		return err
	}

	return nil
}

// Done prints out final success message
func (d *sdFlasher) Done() error {
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
	fmt.Printf("\t\t Please insert your sd card into your %s\n", d.device)
	if d.devRepo.Image.User != "" {
		fmt.Println("\t\t ssh to your board with the following credentials")
		fmt.Printf("\t\t ssh username: "+dialogs.PrintColored("%s")+" password: "+dialogs.PrintColored("%s")+"\n",
			d.devRepo.Image.User, d.devRepo.Image.Pass)
	}
	fmt.Println("\t\t If you have any questions or suggestions feel free to make an issue at https://github.com/xshellinc/iotit/issues/ or tweet us @isaax_iot")

	return nil
}

func (d *sdFlasher) execOverSSH(command string, outp *string) error {
	log.WithField("command", command).Debug("execOverSSH")
	if out, eut, err := d.conf.SSH.Run(command); err != nil {
		log.Error("[-] Error executing: ", command, eut)
		return err
	} else if strings.TrimSpace(eut) != "" {
		out = strings.TrimSpace(out)
		eut = strings.TrimSpace(eut)
		log.WithField("out", out).WithField("eut", eut).Debug("execOverSSH Error")
		if outp != nil {
			*outp = eut
		}
		return fmt.Errorf(eut)
	} else if strings.TrimSpace(out) != "" {
		out = strings.TrimSpace(out)
		eut = strings.TrimSpace(eut)
		log.WithField("out", out).WithField("eut", eut).Debug("execOverSSH Output")
		if outp != nil {
			*outp = out
		}
	}
	return nil
}
