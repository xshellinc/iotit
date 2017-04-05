package device

import (
	"sync"

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
	wg := &sync.WaitGroup{}
	job := help.NewBackgroundJob()

	go func() {
		defer wg.Wait()
		defer job.Close()
		wg.Add(1)

		if err := d.MountImg(beagleMount); err != nil {
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

func (d *beagleBone) Done() error {
	printDoneMessageSd("Nano PI", constants.DEFAULT_BEAGLEBONE_USERNAME, constants.DEFAULT_BEAGLEBONE_PASSWORD)

	return nil
}
