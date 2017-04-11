package device

import (
	"fmt"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

const (
	beagleMount = "p1"
)

// beagleBone device
type beagleBone struct {
	*sdFlasher
}

// Configure overrides sdFlasher Configure() method with custom config
func (d *beagleBone) Configure() error {
	job := help.NewBackgroundJob()
	c := config.NewDefault(d.conf.SSH)

	go func() {
		defer job.Close()

		if err := d.MountImg(beagleMount); err != nil {
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

	logrus.Debug("Linking tmp folder")
	command := fmt.Sprintf("ln -sf %s %s/%s", "/dev/null", filepath.Join(constants.MountDir, "etc", "udev", "rules.d"), "80-net-setup-link.rules")
	out, eut, err := d.conf.SSH.Run(command)
	if err != nil {
		logrus.Error("[-] Error when execute: ", command, eut)
		return err
	}
	logrus.Debug(out, eut)

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
