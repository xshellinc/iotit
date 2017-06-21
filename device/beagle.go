package device

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/dialogs"
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

	if dialogs.YesNoDialog("Would you like to configure your board?") {
		// setup while background process mounting img
		if err := c.Setup(); err != nil {
			return err
		}
	}

	if err := help.WaitJobAndSpin("Waiting", job); err != nil {
		return err
	}
	// why?
	command := fmt.Sprintf("ln -sf %s %s/%s", "/dev/null", help.AddPathSuffix("unix", config.MountDir, "etc", "udev", "rules.d"), "80-net-setup-link.rules")
	log.WithField("command", command).Debug("Linking tmp folder")
	out, eut, err := d.conf.SSH.Run(command)
	if err != nil {
		log.Error("[-] Error when execute: ", command, eut)
		return err
	}
	log.Debug(out, eut)

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
