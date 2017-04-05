package device

import (
	"github.com/pkg/errors"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
)

const BadRepoError = "Bad repository "

var devices = [...]string{
	constants.DEVICE_TYPE_RASPBERRY,
	constants.DEVICE_TYPE_EDISON,
	constants.DEVICE_TYPE_NANOPI,
	constants.DEVICE_TYPE_BEAGLEBONE,
}

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

func selectImage(mapping *repo.DeviceMapping) *repo.DeviceMapping {
	selected := selectDevice(mapping)

	if len(selected.Images) == 0 {
		return nil
	}

	n := 0
	if len(selected.Images) > 1 {
		n = dialogs.SelectOneDialog("Please select an image: ", selected.GetImageTitles())
	}

	selected.Url = selected.Images[n].Url

	return selected
}
