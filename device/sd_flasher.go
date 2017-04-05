package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/workstation"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

type SdFlasher interface {
	MountImg() error
	Config() error
	UnmountImg() error
	Flash() error
	Done() error
}

type sdFlasher struct {
	*deviceFlasher
}

func (d *sdFlasher) MountImg(loopMount string) error {

	logrus.Debug("Attaching an image")
	command := fmt.Sprintf("losetup -f -P %s", filepath.Join(constants.TMP_DIR, d.img))
	out, eut, err := d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out)

	logrus.Debug("Creating tmp folder")
	command = fmt.Sprintf("mkdir -p %s", constants.GENERAL_MOUNT_FOLDER)
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out)

	if loopMount == "" {
		out, eut, err = d.conf.SSH.Run("ls /dev/loop0*")
		if err != nil {
			logrus.Error("[-] Error when execute: ", command, eut)
			return err
		}
		logrus.Debug(out)

		opts := strings.Split(out, "\n")

		n := 0
		if len(opts) > 1 {
			n = dialogs.SelectOneDialog("Select correct mount", opts)
		}

		loopMount = strings.TrimSpace(opts[n])
	}

	logrus.Debug("Mounting tmp folder")
	command = fmt.Sprintf("%s -o rw /dev/loop0%s %s", constants.LINUX_MOUNT, loopMount, constants.GENERAL_MOUNT_FOLDER)
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out)

	logrus.Debug("Linking tmp folder")
	command = fmt.Sprintf("ln -sf %s %s/%s", "/dev/null", filepath.Join(constants.GENERAL_MOUNT_FOLDER, "etc", "udev", "rules.d"), "80-net-setup-link.rules")
	out, eut, err = d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out)

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

	d.conf = vbox.NewConfig(d.device)

	err := d.conf.SSH.ScpFromServer(help.AddPathSuffix(constants.TMP_DIR, d.img, "unix"), filepath.Join(constants.TMP_DIR, d.img))
	if err != nil {
		return err
	}

	w := workstation.NewWorkStation()

	logrus.Debug("Writing the image into sd card")
	job, err := w.WriteToDisk(d.img)
	if err != nil {
		return err
	}
	if job != nil {
		if err := help.WaitJobAndSpin("flashing", job); err != nil {
			return err
		}
	}

	logrus.Debug("Removing sd from dir")
	if err = os.Remove(d.img); err != nil {
		logrus.Error("[-] Can not remove image: " + err.Error())
	}

	err = w.Unmount()
	if err != nil {
		logrus.Error("Error parsing mount option ", "error msg:", err.Error())
	}
	err = w.Eject()
	if err != nil {
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

func (d *sdFlasher) Config() error {
	config := NewSetDevice(d.device)
	err := config.SetConfig()

	if dialogs.YesNoDialog("Add Google DNS as a secondary NameServer") {
		if _, eut, err := d.conf.SSH.Run(fmt.Sprintf(AddGoogleNameServerCmd, constants.GENERAL_MOUNT_FOLDER+"etc/dhcp/dhclient.conf")); err != nil {
			logrus.Error("Error adding google dns ", "error msg:", eut)
			return err
		}
	}

	err = config.Upload(d.conf)

	return err
}

func (d *sdFlasher) Configure() error {
	wg := &sync.WaitGroup{}
	job := help.NewBackgroundJob()

	wg.Add(1)
	go func() {
		fmt.Println("Running command conf sdFlash")
		defer wg.Wait()
		defer job.Close()

		if err := d.MountImg(""); err != nil {
			job.Error(err)
		}
	}()

	if err := d.Config(); err != nil {
		return err
	}

	if err := help.WaitJobAndSpin("waiting", job); err != nil {
		return err
	}
	wg.Wait()

	if err := d.UnmountImg(); err != nil {
		return err
	}
	if err := d.Flash(); err != nil {
		return err
	}

	return d.Done()
}
