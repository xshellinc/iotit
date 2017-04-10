package device

import (
	"fmt"

	"strings"

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

	fmt.Println("[+] flashing", deviceType)

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

// Prints sd card flashed message
func printDoneMessageSd(device, username, password string) {
	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t SD CARD READY!  \t\t\t\t\t\t\t\t   *")
	fmt.Printf("*\t\t PLEASE INSERT YOUR SD CARD TO YOUR %s \t\t\t\t\t   *\n", device)
	fmt.Println("*\t\t IF YOU HAVE NOT SET UP THE USB WIFI, PLEASE CONNECT TO ETHERNET \t\t   *")
	fmt.Printf("*\t\t SSH USERNAME:\x1b[31m%s\x1b[0m PASSWORD:\x1b[31m%s\x1b[0m \t\t\t\t\t\t\t   *\n",
		username, password)
	fmt.Println(strings.Repeat("*", 100))
}
