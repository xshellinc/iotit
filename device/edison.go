package device

import (
	"fmt"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/lib/help"
	"os"
	"github.com/xshellinc/tools/constants"
)

const (
	baseConf  string = "base-feeds.conf"
	iotdkConf string = "intel-iotdk.conf"

	baseFeeds string = "src/gz all        http://repo.opkg.net/edison/repo/all\n" +
		"src/gz edison     http://repo.opkg.net/edison/repo/edison\n" +
		"src/gz core2-32   http://repo.opkg.net/edison/repo/core2-32\n"

	intelIotdk string = "src intel-all     http://iotdk.intel.com/repos/1.1/iotdk/all\n" +
		"src intel-iotdk   http://iotdk.intel.com/repos/1.1/intelgalactic\n" +
		"src intel-quark   http://iotdk.intel.com/repos/1.1/iotdk/quark\n" +
		"src intel-i586    http://iotdk.intel.com/repos/1.1/iotdk/i586\n" +
		"src intel-x86     http://iotdk.intel.com/repos/1.1/iotdk/x86\n"
)

type edison struct {
	*deviceFlasher
}

func (d *edison) PrepareVbox() error {
	ack := dialogs.YesNoDialog("Would you like to flash your device? ")

	if ack {
		return d.deviceFlasher.PrepareVbox()
	}

	return nil
}

func (d *edison) Configure() error {
	for !dialogs.YesNoDialog("Please unplug your edison board. Press yes once unpluged? ") {}

	for {
		script := "flashall.sh"

		args := []string{
			fmt.Sprintf("%s@%s", vbox.VBoxUser, vbox.VBoxIP),
			"-p",
			vbox.VBoxSSHPort,
			constants.TMP_DIR + script,
		}

		if err := help.ExecStandardStd("ssh", args...); err != nil {
			fmt.Println("[-] Cannot find mounted Intel edison device, please try to re-mount it")

			if !dialogs.YesNoDialog("Press yes once mounted? ") {
				fmt.Println("Exiting with exit status 2 ...")
				os.Exit(2)
			}

			continue
		}

		break
	}

	return nil
}
