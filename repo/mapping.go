package repo

import (
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/tools/lib/help"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeviceImage contains url, title, username and password which are used after flashing
type DeviceImage struct {
	URL   string `json:"URL"`
	Alias string `json:"Alias,omitempty"`
	Title string `json:"Title,omitempty"`
	User  string `json:"User,omitempty"`
	Pass  string `json:"Pass,omitempty"`
	Hash  string `json:"Hash,omitempty"`
}

// DeviceMapping is a collection of device, it sub-types and sets of images for these devices
type DeviceMapping struct {
	Name   string           `json:"Name"`
	Alias  string           `json:"Alias,omitempty"`
	Sub    []*DeviceMapping `json:"Sub,omitempty"`
	Images []DeviceImage    `json:"Images,omitempty"`
	Type   string

	dir   string
	Image DeviceImage
}

// deviceCollection is a starting point of the collection of images
type DeviceCollection struct {
	Devices []DeviceMapping `json:"Devices"`
	Version string          `json:"Version,omitempty"`
}

// Images repository URL
const imagesRepo = "https://isaaxartifacts.blob.core.windows.net/iotit/mapping.json"

var path string
var dm *DeviceCollection

func init() {
	path = filepath.Join(baseDir, "mapping.json")
}

// GetSubsNames returns array of Names within a `Sub`
func (d *DeviceMapping) GetSubsNames() []string {
	r := make([]string, len(d.Sub))
	for i, o := range d.Sub {
		r[i] = o.Name
	}

	return r
}

// GetImageTitles returns array of image titles
func (d *DeviceMapping) GetImageTitles() []string {
	r := make([]string, len(d.Images))
	for i, o := range d.Images {
		r[i] = o.Title
	}

	return r
}

// Dir - returns directory of a local repo `.iotit/images/{name}`
func (d *DeviceMapping) Dir() string {
	return filepath.Join(ImageDir, d.dir)
}

// FindImage - searches image in the repo
func (d *DeviceMapping) FindImage(image string) error {
	search := strings.ToLower(image)
	for _, obj := range d.Images {
		if strings.ToLower(obj.Title) == search || obj.Alias == search {
			d.Image = obj
			return nil
		}
	}

	return errors.New(image + " unknown image")
}

// findDevice searches device in the repo
func (d *DeviceCollection) findDevice(device string) (*DeviceMapping, error) {
	search := strings.ToLower(device)
	for _, obj := range d.Devices {
		obj.dir = obj.Name
		obj.Type = obj.Name
		if strings.ToLower(obj.Name) == search || obj.Alias == search {
			fillEmptyImages(&obj)
			return &obj, nil
		}
		if len(obj.Sub) > 0 {
			for _, sub := range obj.Sub {
				sub.dir = obj.Name
				sub.Type = obj.Name
				if strings.ToLower(sub.Name) == search || sub.Alias == search {
					if len(sub.Images) == 0 {
						sub.Images = obj.Images
					}
					return sub, nil
				}
			}
		}
	}

	return nil, errors.New(device + " device is not supported")
}

func (d *DeviceCollection) getDevices() []string {
	devices := []string{}
	for _, obj := range d.Devices {
		devices = append(devices, obj.Name)
	}
	return devices
}

// DownloadDevicesRepository downloads new mapping.json from the cloud
func DownloadDevicesRepository() error {
	log.Info("Downloading new mapping.json...")
	if err := help.DownloadFile(path, imagesRepo); err != nil {
		log.Error(err)
		return err
	}
	// update file modification date so on the next run we don't try to download it again
	if err := os.Chtimes(path, time.Now(), time.Now()); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

// CheckDevicesRepository checks mapping.json for updates
func CheckDevicesRepository() {
	log.Info("Checking for mapping.json updates...")
	if info, err := os.Stat(path); os.IsNotExist(err) || time.Now().Sub(info.ModTime()).Hours() >= 24 {
		DownloadDevicesRepository()
	}
}

// SetPath of the mapping.json file
func SetPath(p string) {
	path = p
}

// GetAllRepos returns all repo Names
func GetAllRepos() ([]string, error) {
	if dm == nil {
		if err := initDeviceCollection(); err != nil {
			return nil, err
		}
	}

	str := make([]string, 0)

	for _, d := range dm.Devices {
		str = append(str, d.Name)
	}

	return str, nil
}

// GetRepo returns devices collection
func GetRepo() *DeviceCollection {
	if dm == nil {
		if err := initDeviceCollection(); err != nil {
			return &DeviceCollection{}
		}
	}
	return dm
}

// GetDeviceRepo returns a devices repo. It checks the existence of mapping.json first then proceeds to the default variable
func GetDeviceRepo(device string) (*DeviceMapping, error) {
	if dm == nil {
		if err := initDeviceCollection(); err != nil {
			return nil, err
		}
	}

	return dm.findDevice(device)
}

func GetDevices() []string {
	if dm == nil {
		if err := initDeviceCollection(); err != nil {
			return []string{}
		}
	}
	return dm.getDevices()
}

// initDeviceCollection initializes deviceCollection from file, alternatively from the internal constant
func initDeviceCollection() error {
	dm = &DeviceCollection{}

	d, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(d, dm); err != nil {
		return err
	}

	return nil
}

// GenMappingFile generates mapping.json file
func GenMappingFile() error {
	DownloadDevicesRepository()
	return nil
}

// fillEmptyImages updates substructures' empty image arrays with the parent one
func fillEmptyImages(m *DeviceMapping) {
	if len(m.Images) != 0 {
		for _, dm := range m.Sub {
			dm.dir = m.Name
			if len(dm.Images) == 0 {
				dm.Images = m.Images
			}

			fillEmptyImages(dm)
		}
	}
}
