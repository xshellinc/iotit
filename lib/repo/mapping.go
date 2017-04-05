package repo

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

const missingRepo = "Device repo is missing"

var path string

func init() {
	p, err := os.Executable()
	if err != nil {
		logrus.Error(err)
	}

	path = filepath.Join(p, file)
}

type DeviceImage struct {
	Url   string `json`
	Title string `json`
}

type DeviceMapping struct {
	Name   string           `json`
	Sub    []*DeviceMapping `json`
	Images []DeviceImage    `json`

	dir string
	Url string
}

func (d *DeviceMapping) GetSubsNames() []string {
	r := make([]string, len(d.Sub))
	for i, o := range d.Sub {
		r[i] = o.Name
	}

	return r
}

func (d *DeviceMapping) GetImageTitles() []string {
	r := make([]string, len(d.Images))
	for i, o := range d.Images {
		r[i] = o.Title
	}

	return r
}

func (d *DeviceMapping) Dir() string {
	return filepath.Join(ImageDir, d.dir)
}

type deviceCollection struct {
	Devices []*DeviceMapping `json`
}

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

func SetPath(p string) {
	path = p
}

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

func GenMappingFile() error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(example)

	return nil
}

func IsMissingRepoError(err error) bool {
	if err.Error() == missingRepo {
		return true
	}

	return false
}

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
