package repo

import (
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/tools/lib/help"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DeviceImage contains url, title, username and password which are used after flashing
type DeviceImage struct {
	URL   string `json:"URL"`
	Alias string `json:"Alias,omitempty"`
	Title string `json:"Title,omitempty"`
	User  string `json:"User,omitempty"`
	Pass  string `json:"Pass,omitempty"`
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
const imagesRepo = "https://cdn.isaax.io/iotit/mapping.json"

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
					return sub, nil
				}
			}
		}
	}

	return nil, errors.New(device + " device is not supported")
}

// DownloadDevicesRepository downloads new mapping.json from the cloud
func DownloadDevicesRepository() {
	if info, err := os.Stat(path); os.IsNotExist(err) || time.Now().Sub(info.ModTime()).Hours() >= 24 {
		log.Info("Checking for mapping.json updates...")
		wg := &sync.WaitGroup{}
		_, _, err := help.DownloadFromUrlWithAttemptsAsync(imagesRepo, baseDir, 3, wg)
		if err != nil {
			log.Error(err)
		}
		wg.Wait()
		// update file modification date so on the next run we don't try to download it again
		if err := os.Chtimes(path, time.Now(), time.Now()); err != nil {
			log.Error(err)
		}
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
