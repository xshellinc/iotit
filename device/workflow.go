package device

import (
	"github.com/pkg/errors"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
)

// BadRepoError is an error message
const BadRepoError = "Bad repository "

// CustomFlash custom method enum
const CustomFlash = "Custom"

// devices is a list of currently supported devices
var devices = [...]string{
	constants.DEVICE_TYPE_RASPBERRY,
	constants.DEVICE_TYPE_EDISON,
	constants.DEVICE_TYPE_NANOPI,
	constants.DEVICE_TYPE_BEAGLEBONE,
	CustomFlash,
}

// New triggers select repository methods and initializes a new flasher
func New(device string) (Flasher, error) {
	if device == CustomFlash {
		r, err := repo.GetAllRepos()
		if err != nil {
			return nil, err
		}

		g := make([]string, 0)
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

		if len(g) == 0 {
			return nil, errors.New("No custom boards are available")
		}

		device = g[dialogs.SelectOneDialog("Please select a cutom board", g)]

	}

	r, err := repo.GetDeviceRepo(device)
	if err != nil && !repo.IsMissingRepoError(err) {
		return nil, err
	}

	if r = selectImage(r); r == nil {
		return nil, errors.New(BadRepoError + device)
	}

	switch device {
	case constants.DEVICE_TYPE_NANOPI:
		i := &nanoPi{&sdFlasher{flasher: &flasher{}}}
		i.device = device
		i.devRepo = r
		return i, err
	case constants.DEVICE_TYPE_RASPBERRY:
		i := &raspberryPi{&sdFlasher{flasher: &flasher{}}}
		i.device = device
		i.devRepo = r
		return i, err
	case constants.DEVICE_TYPE_BEAGLEBONE:
		i := &beagleBone{&sdFlasher{flasher: &flasher{}}}
		i.device = device
		i.devRepo = r
		return i, err
	case constants.DEVICE_TYPE_EDISON:
		i := &edison{&flasher{}}
		i.device = device
		i.devRepo = r
		return i, nil
	default:
		i := &sdFlasher{flasher: &flasher{}}
		i.device = device
		return i, nil
	}
}

// selectDevice is a dialog to select a device if more than one, recursive function
func selectDevice(mapping *repo.DeviceMapping) *repo.DeviceMapping {
	var selected *repo.DeviceMapping
	if len(mapping.Sub) > 1 {
		n := dialogs.SelectOneDialog("Please select a device type: ", mapping.GetSubsNames())
		selected = mapping.Sub[n]
	} else if len(mapping.Sub) == 1 {
		selected = mapping.Sub[0]
	} else {
		return mapping
	}
	return selectDevice(selected)
}

// selectImage is a dialog to select an image from the list if more than one, null is returned if nothing is to return
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
