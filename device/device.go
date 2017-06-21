package device

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// badRepoError is an error message
const badRepoError = "Bad repository "

// CustomFlash custom method enum
const customFlash = "custom board"

// devices is a list of currently supported devices
var devices = [...]string{
	constants.DEVICE_TYPE_RASPBERRY,
	constants.DEVICE_TYPE_EDISON,
	constants.DEVICE_TYPE_NANOPI,
	constants.DEVICE_TYPE_BEAGLEBONE,
	constants.DEVICE_TYPE_ESP,
	customFlash,
}

// Flash starts flashing process, either by receiving `typeFlag` or asking user to choose from a list
func Flash(typeFlag string) {
	log.WithField("type", typeFlag).Info("DeviceInit")

	//once in 24h update mapping json
	repo.DownloadDevicesRepository()

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

	flasher, err := getFlasher(deviceType)
	if err != nil {
		fmt.Println("[-] Error: ", err)
		return
	}

	if err := flasher.Flash(); err != nil {
		fmt.Println("[-] Error: ", err)
		return
	}
}

// getFlasher triggers select repository methods and initializes a new flasher
func getFlasher(device string) (Flasher, error) {
	g := make([]string, 0)

	if device == customFlash {
		r, err := repo.GetAllRepos()
		if err != nil {
			return nil, err
		}

		for _, s := range r {
			c := true
			for _, d := range devices {
				if s == d {
					c = false
				}
			}
			if c {
				g = append(g, s)
			}
		}

		if len(g) != 0 {
			device = g[dialogs.SelectOneDialog("Please select your custom board: ", g)]
		}

	}
	var r *repo.DeviceMapping
	if device == customFlash && len(g) == 0 {
		fmt.Println("[-] No custom boards defined")

		url := dialogs.GetSingleAnswer("Please provide image URL or path: ", dialogs.EmptyStringValidator)
		r = &repo.DeviceMapping{Name: "Custom", Image: repo.DeviceImage{URL: url}}
	} else {
		var e error
		r, e = repo.GetDeviceRepo(device)

		if e != nil {
			return nil, e
		}

		if r = selectImage(r); r == nil {
			return nil, errors.New(badRepoError + device)
		}
	}

	switch device {
	case constants.DEVICE_TYPE_RASPBERRY:
		i := &raspberryPi{&sdFlasher{flasher: &flasher{}}}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_BEAGLEBONE:
		i := &beagleBone{&sdFlasher{flasher: &flasher{}}}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_EDISON:
		i := &edison{flasher: &flasher{}}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_ESP:
		i := &serialFlasher{flasher: &flasher{}}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_NANOPI:
		fallthrough
	default:
		i := &sdFlasher{flasher: &flasher{}}
		i.device = device
		i.devRepo = r
		return i, nil
	}
}

// selectDevice is a dialog to select a device if more than one, recursive function
func selectDevice(mapping *repo.DeviceMapping) *repo.DeviceMapping {
	var selected *repo.DeviceMapping
	if len(mapping.Sub) > 1 {
		n := dialogs.SelectOneDialog("Please select device type: ", mapping.GetSubsNames())
		selected = mapping.Sub[n]
	} else if len(mapping.Sub) == 1 {
		selected = mapping.Sub[0]
	} else {
		return mapping
	}
	return selectDevice(selected)
}

// selectImage is a dialog to select an image from the list if more than one, nil is returned if nothing is to return
func selectImage(mapping *repo.DeviceMapping) *repo.DeviceMapping {
	selected := selectDevice(mapping)

	if len(selected.Images) == 0 {
		return nil
	}

	n := 0
	if len(selected.Images) > 1 {
		n = dialogs.SelectOneDialog("Please select an image: ", selected.GetImageTitles())
	}

	selected.Image = selected.Images[n]

	return selected
}
