// +build !windows

package device

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/vbox"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

func (d *edison) Prepare() error {
	if err := d.flasher.Prepare(); err != nil {
		return err
	}

	for !dialogs.YesNoDialog("Please unplug your Edison board. Type yes once unpluged.") {
	}
	return nil
}

func (d *edison) Write() error {
	for {
		script := "flashall.sh"
		args := []string{
			fmt.Sprintf("%s@%s", vbox.VBoxUser, vbox.VBoxIP),
			"-p",
			vbox.VBoxSSHPort,
			config.TmpDir + script,
		}
		if err := help.ExecStandardStd("ssh", args...); err != nil {
			fmt.Println("[-] Can't find Intel Edison board, please try to re-connect it")

			if !dialogs.YesNoDialog("Type yes once connected.") {
				fmt.Println("Exiting with exit status 2 ...")
				os.Exit(2)
			}
			continue
		}
		break
	}

	if err := d.conf.Stop(d.Quiet); err != nil {
		log.Error(err)
	}

	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		time.Sleep(120 * time.Second)
	}()

	help.WaitJobAndSpin("Your Edison board is restarting...", job)
	return nil
}
