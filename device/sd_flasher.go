package device

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/device/workstation"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

type SdFlasher interface {
	MountImg() error
	UnmountImg() error
	Flash() error
	Done() error
}

type sdFlasher struct {
	*deviceFlasher
}

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
	command = fmt.Sprintf("mkdir -p %s", constants.GENERAL_MOUNT_FOLDER)
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out, eut)

	logrus.Debug("Mounting tmp folder")
	command = fmt.Sprintf("%s -o rw /dev/loop0%s %s", constants.LINUX_MOUNT, loopMount, constants.GENERAL_MOUNT_FOLDER)
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out, eut)

	logrus.Debug("Linking tmp folder")
	command = fmt.Sprintf("ln -sf %s %s/%s", "/dev/null", filepath.Join(constants.GENERAL_MOUNT_FOLDER, "etc", "udev", "rules.d"), "80-net-setup-link.rules")
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out, eut)

	return nil
}

func (d *sdFlasher) UnmountImg() error {
	logrus.Debug("Unlinking tmp folder")
	command := fmt.Sprintf("umount %s", constants.GENERAL_MOUNT_FOLDER)
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

func (d *sdFlasher) Done() error {
	fmt.Println("Fashing is completed")
	return nil
}

func (d *sdFlasher) Configure() error {
	job := help.NewBackgroundJob()
	c := config.NewDefault(d.conf.SSH)

	go func() {
		defer job.Close()

		if err := d.MountImg(""); err != nil {
			job.Error(err)
		}
	}()

	// setup while background process mounting img
	if err := c.Setup(); err != nil {
		return err
	}

	if err := help.WaitJobAndSpin("waiting", job); err != nil {
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
