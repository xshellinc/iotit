package device

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// Init starts init process, either by receiving `typeFlag` or providing a user to choose from a list
func Init(typeFlag string) {
	log.Info("DeviceInit")
	log.Debug("Flag: ", typeFlag)

	var deviceType string

	if typeFlag != "" {
		if help.StringToSlice(typeFlag, devices[:]) {
			deviceType = typeFlag
		} else {
			fmt.Println("[-]", typeFlag, "device is not supported")
		}
	}

	if deviceType == "" {
		deviceType = devices[dialogs.SelectOneDialog("Select device type: ", devices[:])]
	}

	fmt.Println("[+] Flashing", deviceType)

	df, err := New(deviceType)
	if err != nil {
		fmt.Println("[-] Error: ", err)
		return
	}
	if err := df.PrepareForFlashing(); err != nil {
		fmt.Println("[-] Error: ", err)
		return
	}
	if err := df.Configure(); err != nil {
		fmt.Println("[-] Error: ", err)
		return
	}
}
