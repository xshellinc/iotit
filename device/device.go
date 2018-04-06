package device

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/iotit/repo"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"gopkg.in/urfave/cli.v1"
)

// CustomFlash custom method enum
const customFlash = "Custom board"

// New returns new Flasher instance
func New(c *cli.Context) Flasher {
	args := c.Args()[:]

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
	repo.CheckDevicesRepository()

	var deviceType string

	if len(typeArg) > 0 {
		if d, err := repo.GetDeviceRepo(typeArg); err != nil {
			help.ExitOnError(err)
		} else {
			deviceType = d.Name
		}
	} else {
		deviceNames := repo.GetDevices()
		deviceNames = append(deviceNames, customFlash)
		deviceType = deviceNames[dialogs.SelectOneDialog("Select device type: ", deviceNames)]
	}

	fmt.Println("[+] Flashing", deviceType)

	flasher, err := getFlasher(deviceType, imgArg, c)
	if err != nil {
		fmt.Println("[-] Error: ", err)
		return nil
	}
	return flasher
}

// ListItem is an item in supported devices list
type ListItem struct {
	Title  string
	Alias  string
	Images map[string]string
	Models []ListItem
}

// ListMapping - print supported devices from mapping.json file
func ListMapping() []*ListItem {
	list := []*ListItem{}
	dm := repo.GetRepo()
	fmt.Println("mapping.json version:", dm.Version)
	for _, device := range dm.Devices {
		r, e := repo.GetDeviceRepo(device.Name)
		if e != nil {
			continue
		}
		item := ListItem{}
		item.Title = r.Name
		if len(r.Alias) > 0 {
			item.Alias = r.Alias
		}
		if len(r.Sub) == 0 {
			item.Images = make(map[string]string)
			for _, i := range r.Images {
				item.Images[i.Title] = i.Alias
			}
		} else {
			for _, sub := range r.Sub {
				model := ListItem{}
				model.Title = sub.Name
				if len(sub.Alias) > 0 {
					model.Alias = sub.Alias
				}
				model.Images = make(map[string]string)
				for _, i := range sub.Images {
					model.Images[i.Title] = i.Alias
				}
				item.Models = append(item.Models, model)
			}
		}
		list = append(list, &item)
	}
	return list
}

// getFlasher triggers select repository methods and initializes a new flasher
func getFlasher(device, image string, c *cli.Context) (Flasher, error) {
	var r *repo.DeviceMapping

	port := c.String("port")
	disk := c.String("disk")
	quiet := c.Bool("quiet")

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

	if r.Type == "" {
		r.Type = device
	}

	switch r.Type {
	case "Raspberry Pi":
		i := &raspberryPi{&sdFlasher{flasher: &flasher{Quiet: quiet, CLI: c}, Disk: disk}}
		i.device = device
		i.devRepo = r
		return i, nil
	case "Beaglebone":
		i := &beagleBone{&sdFlasher{flasher: &flasher{Quiet: quiet, CLI: c}, Disk: disk}}
		i.device = device
		i.devRepo = r
		return i, nil
	case "Toradex Colibri iMX6":
		i := &colibri{&flasher{Quiet: quiet, CLI: c}, port, disk}
		i.device = device
		i.devRepo = r
		return i, nil
	case "Intel® Edison":
		i := &edison{flasher: &flasher{Quiet: quiet, CLI: c}, IP: port}
		i.device = device
		i.devRepo = r
		return i, nil
	case "Espressif ESP":
		i := &serialFlasher{&flasher{Quiet: quiet, CLI: c}, port}
		i.device = device
		i.devRepo = r
		return i, nil
	case "Nano Pi":
		fallthrough
	case "ASUS Tinker Board":
		fallthrough
	default:
		i := &sdFlasher{flasher: &flasher{Quiet: quiet, CLI: c}, Disk: disk}
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
