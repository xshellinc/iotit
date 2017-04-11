package repo

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

const missingRepo = "Device repo is missing"

var path string

func init() {
	//p, err := os.Executable()
	//if err != nil {
	//	logrus.Error(err)
	//}

	path = filepath.Join(baseDir, file)
}

// DeviceImage contains url, title, username and password which are used after flashing
type DeviceImage struct {
	URL   string `json:"URL"`
	Title string `json:"Title,omitempty"`
	User  string `json:"User,omitempty"`
	Pass  string `json:"Pass,omitempty"`
}

// DeviceMapping is a collection of device, it sub-types and sets of images for these devices
type DeviceMapping struct {
	Name   string           `json:"Name"`
	Sub    []*DeviceMapping `json:"Sub,omitempty"`
	Images []DeviceImage    `json:"Images,omitempty"`

	dir   string
	Image DeviceImage
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

// Dir returns directory of a local repo `.iotit/images/{name}`
func (d *DeviceMapping) Dir() string {
	return filepath.Join(ImageDir, d.dir)
}

// deviceCollection is a starting point of the collection of images
type deviceCollection struct {
	Devices []*DeviceMapping `json:"Devices"`
}

// findDevice searches device in the repo
func (d *deviceCollection) findDevice(device string) (*DeviceMapping, error) {
	for _, obj := range d.Devices {
		obj.dir = obj.Name
		if obj.Name == device {
			fillEmptyImages(obj)
			return obj, nil
		}
	}

	return nil, errors.New(missingRepo)
}

// SetPath of the mapping.json file
func SetPath(p string) {
	path = p
}

// GetDeviceRepo returns a devices repo. It checks the existance of mapping.json first then proceeds to the default variable
func GetDeviceRepo(device string) (*DeviceMapping, error) {
	dm := deviceCollection{}

	if _, err := os.Stat(path); err != nil {
		if os.IsExist(err) {
			return nil, err
		}

		if err := json.Unmarshal([]byte(example), &dm); err != nil {
			return nil, err
		}
	} else {
		d, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(d, &dm); err != nil {
			return nil, err
		}
	}

	return dm.findDevice(device)
}

// GenMappingFile generates mapping.json file
func GenMappingFile() error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(example)

	return nil
}

// IsMissingRepoError checks error if error is caused by missingRepo
func IsMissingRepoError(err error) bool {
	if err.Error() == missingRepo {
		return true
	}

	return false
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
