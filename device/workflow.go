package device

import (
	"github.com/xshellinc/tools/constants"
)

var devices = [...]string{
	constants.DEVICE_TYPE_RASPBERRY,
	constants.DEVICE_TYPE_EDISON,
	constants.DEVICE_TYPE_NANOPI,
	constants.DEVICE_TYPE_BEAGLEBONE,
}

func New(device string) DeviceFlasher {
	switch device {
	case constants.DEVICE_TYPE_NANOPI:
		i := &nanoPi{&sdFlasher{&deviceFlasher{}, nil}}
		i.device = device
		return i
	case constants.DEVICE_TYPE_RASPBERRY:
		i := &raspberryPi{&sdFlasher{&deviceFlasher{}, nil}}
		i.device = device
		return i
	case constants.DEVICE_TYPE_BEAGLEBONE:
		i := &beagleBone{&sdFlasher{&deviceFlasher{}, nil}}
		i.device = device
		return i
	case constants.DEVICE_TYPE_EDISON:
		i := &edison{&deviceFlasher{}}
		i.device = device
		return i
	default:
		i := &sdFlasher{&deviceFlasher{}, nil}
		i.device = device
		return i
	}
}



