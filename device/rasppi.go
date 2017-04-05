package device

import (
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

const (
	raspMount = "p2"
)

type raspberryPi struct {
	*sdFlasher
}

func (d *raspberryPi) Configure() error {
	job := help.NewBackgroundJob()

	go func() {
		defer job.Close()

		if err := d.MountImg(raspMount); err != nil {
			job.Error(err)
		}
	}()

	if err := d.Config(); err != nil {
		return err
	}

	if err := help.WaitJobAndSpin("waiting", job); err != nil {
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

func (d *raspberryPi) Done() error {
	printDoneMessageSd("RASPBERRY PI", constants.DEFAULT_RASPBERRY_USERNAME, constants.DEFAULT_RASPBERRY_PASSWORD)

	return nil
}
