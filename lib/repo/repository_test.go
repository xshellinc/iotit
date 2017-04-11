package repo

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xshellinc/tools/constants"
)

func init() {
	path = "/tmp/mapper.json"
}

func cleanUp() {
	os.Remove(path)
}

func TestGenMappingFile(t *testing.T) {
	assert := assert.New(t)
	err := GenMappingFile()
	assert.NoError(err)

	stat, err := os.Stat(path)
	assert.NoError(err)
	assert.Equal(stat.Mode()&0644, os.FileMode(0644), "Wrong filemode:", stat.Mode())

	cleanUp()
}

func TestGetDeviceRepo(t *testing.T) {
	assert := assert.New(t)
	dev, err := GetDeviceRepo(constants.DEVICE_TYPE_RASPBERRY)
	assert.NoError(err)

	assert.Equal(dev.Name, constants.DEVICE_TYPE_RASPBERRY)
	assert.EqualValues(dev.Images, dev.Sub[0].Images)

	err = GenMappingFile()
	assert.NoError(err)

	dev, err = GetDeviceRepo(constants.DEVICE_TYPE_RASPBERRY)
	assert.NoError(err)

	assert.Equal(dev.Name, constants.DEVICE_TYPE_RASPBERRY)
	assert.EqualValues(dev.Images, dev.Sub[0].Images)

	cleanUp()
}

func TestGetVersionLexem(t *testing.T) {
	assert := assert.New(t)

	assert.Equal([]string{"1", "15", "2"}, getVersionLexem("1.15.2", "."))
	assert.Equal([]string{"1", "15", "4", "2"}, getVersionLexem("1.15_4.2", ".", "_"))
}
