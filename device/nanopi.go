package device

import (
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

const (
	nanoMount = "p2"
)

type nanoPi struct {
	*sdFlasher
}

func (d *nanoPi) Configure() error {
	job := help.NewBackgroundJob()
	c := config.NewDefault(d.conf.SSH)

	go func() {
		defer job.Close()

		if err := d.MountImg("p2"); err != nil {
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

func (d *nanoPi) Done() error {
	printDoneMessageSd(d.device, constants.DEFAULT_NANOPI_USERNAME, constants.DEFAULT_NANOPI_PASSWORD)

	return nil
}
