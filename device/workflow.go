package device

import (
	"github.com/pkg/errors"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
)

// BadRepoError is an error message
const BadRepoError = "Bad repository "

// devices is a list of currently supported devices
var devices = [...]string{
	constants.DEVICE_TYPE_RASPBERRY,
	constants.DEVICE_TYPE_EDISON,
	constants.DEVICE_TYPE_NANOPI,
	constants.DEVICE_TYPE_BEAGLEBONE,
}

// New triggers select repository methods and initializes a new deviceFlasher
func New(device string) (DeviceFlasher, error) {
	r, err := repo.GetDeviceRepo(device)
	if err != nil && !repo.IsMissingRepoError(err) {
		return nil, err
	}

	if r = selectImage(r); r == nil {
		return nil, errors.New(BadRepoError + device)
	}

	switch device {
	case constants.DEVICE_TYPE_NANOPI:
		i := &nanoPi{&sdFlasher{deviceFlasher: &deviceFlasher{}}}
		i.device = device
		i.devRepo = r
		return i, err
	case constants.DEVICE_TYPE_RASPBERRY:
		i := &raspberryPi{&sdFlasher{deviceFlasher: &deviceFlasher{}}}
		i.device = device
		i.devRepo = r
		return i, err
	case constants.DEVICE_TYPE_BEAGLEBONE:
		i := &beagleBone{&sdFlasher{deviceFlasher: &deviceFlasher{}}}
		i.device = device
		i.devRepo = r
		return i, err
	case constants.DEVICE_TYPE_EDISON:
		i := &edison{&deviceFlasher{}}
		i.device = device
		i.devRepo = r
		return i, nil
	default:
		i := &sdFlasher{deviceFlasher: &deviceFlasher{}}
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

	selected.Url = selected.Images[n]

	return selected
}
