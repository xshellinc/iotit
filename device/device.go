package device

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

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

// Flash starts flashing process
func Flash(args []string, port string, quiet bool) {
	log.WithField("args", args).Info("DeviceInit")
	typeArg := ""
	imgArg := ""
	if len(args) > 0 {
		typeArg = args[0]
	}
	if len(args) > 1 {
		imgArg = args[1]
	}

	//once in 24h update mapping json
	repo.DownloadDevicesRepository()

	var deviceType string

	if len(typeArg) > 0 {
		if d, err := repo.GetDeviceRepo(typeArg); err != nil {
			help.ExitOnError(err)
		} else {
			deviceType = d.Name
		}
	} else {
		deviceType = devices[dialogs.SelectOneDialog("Select device type: ", devices[:])]
	}

	fmt.Println("[+] Flashing", deviceType)

	flasher, err := getFlasher(deviceType, imgArg, port, quiet)
	if err != nil {
		fmt.Println("[-] Error: ", err)
		return
	}

	if err := flasher.Flash(); err != nil {
		fmt.Println("[-] Error: ", err)
		return
	}
}

func ListMapping() {
	list := make(map[string]interface{})
	dm := repo.GetRepo()
	fmt.Println("mapping.json version:", dm.Version)
	fmt.Println("Devices and images listed as \"name (" + dialogs.PrintColored("alias") + ")\"")
	for _, device := range dm.Devices {
		r, e := repo.GetDeviceRepo(device.Name)
		if e != nil {
			continue
		}
		fmt.Print("Type: " + r.Name)
		if len(r.Alias) > 0 {
			fmt.Print(" (" + dialogs.PrintColored(r.Alias) + ")")
		}
		fmt.Println()
		if len(r.Sub) == 0 {
			fmt.Print("\tImages: ")
			for _, i := range r.Images {
				fmt.Print(i.Title)
				if len(i.Alias) > 0 {
					fmt.Print(" (" + dialogs.PrintColored(i.Alias) + ") ")
				}
			}
			fmt.Println()
		} else {
			for _, sub := range r.Sub {
				fmt.Print("\tModel: " + sub.Name)
				if len(sub.Alias) > 0 {
					fmt.Print(" (" + dialogs.PrintColored(sub.Alias) + ")")
				}
				fmt.Println()
				fmt.Print("\t\tImages: ")
				for _, i := range sub.Images {
					fmt.Print(i.Title)
					if len(i.Alias) > 0 {
						fmt.Print(" (" + dialogs.PrintColored(i.Alias) + ") ")
					}
				}
				fmt.Println()
			}
		}
		list[device.Name+"["+dialogs.PrintColored(device.Alias)+"]"] = *r
	}
}

// getFlasher triggers select repository methods and initializes a new flasher
func getFlasher(device, image, port string, quiet bool) (Flasher, error) {
	var r *repo.DeviceMapping

	if device == customFlash {
		url := dialogs.GetSingleAnswer("Please provide image URL or path: ", dialogs.EmptyStringValidator)
		r = &repo.DeviceMapping{Name: "Custom", Image: repo.DeviceImage{URL: url}}
	} else {
		var e error
		r, e = repo.GetDeviceRepo(device)

		if e != nil {
			return nil, e
		}
		if len(image) > 0 {
			if err := r.FindImage(image); err != nil {
				help.ExitOnError(err)
			}
		}
		if len(r.Image.URL) == 0 {
			if r = selectImage(r); r == nil {
				return nil, errors.New("Empty repository for " + device)
			}
		}
		fmt.Println("[+] Using", r.Image.Title)
	}

	switch r.Type {
	case constants.DEVICE_TYPE_RASPBERRY:
		i := &raspberryPi{&sdFlasher{&flasher{Quiet: quiet}, port}}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_BEAGLEBONE:
		i := &beagleBone{&sdFlasher{&flasher{Quiet: quiet}, port}}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_EDISON:
		i := &edison{flasher: &flasher{Quiet: quiet}, IP: port}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_ESP:
		i := &serialFlasher{&flasher{Quiet: quiet}, port}
		i.device = device
		i.devRepo = r
		return i, nil
	case constants.DEVICE_TYPE_NANOPI:
		fallthrough
	default:
		i := &sdFlasher{&flasher{Quiet: quiet}, port}
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
