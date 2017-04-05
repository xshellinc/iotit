package device

import (
	"sync"

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
	wg := &sync.WaitGroup{}
	job := help.NewBackgroundJob()

	go func() {
		defer wg.Wait()
		defer job.Close()
		wg.Add(1)

		if err := d.MountImg(nanoMount); err != nil {
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

func (d *nanoPi) Done() error {
	printDoneMessageSd("Nano PI", constants.DEFAULT_NANOPI_USERNAME, constants.DEFAULT_NANOPI_PASSWORD)

	return nil
}
