package device

import (
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

const (
	beagleMount = "p1"
)

type beagleBone struct {
	*sdFlasher
}

func (d *beagleBone) Configure() error {
	job := help.NewBackgroundJob()
	c := config.NewDefault(d.conf.SSH)

	go func() {
		defer job.Close()

		if err := d.MountImg("p1"); err != nil {
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

func (d *beagleBone) Done() error {
	printDoneMessageSd(d.device, constants.DEFAULT_BEAGLEBONE_USERNAME, constants.DEFAULT_BEAGLEBONE_PASSWORD)

	return nil
}
