package workstation

import (
	"fmt"
	"strings"
)

type WorkStation interface {
	Check(string) error
	ListRemovableDisk() error
	Unmount() error
	WriteToDisk(img string) (err error, done chan bool)
	Eject() error
}

type workstation struct {
	os      string
	boolean bool
	mount   *MountInfo
	mounts  []*MountInfo
}

// shared type linux/darwin
type unix struct {
	dd      string
	folder  string
	unmount string
	eject   string
}

type MountInfo struct {
	deviceName  string
	diskName    string
	diskNameRaw string
	deviceSize  string
}

func NewWorkStation() WorkStation {

	return newWorkstation()
}

func (m *MountInfo) String() string {
	return fmt.Sprintf("DiskName=%s\n\tdeviceName=%s\n\tdiskNameRaw=%s\n\tdeviceSize=%s",
		m.diskName, m.deviceName, m.diskNameRaw, m.deviceSize)
}

func verify() bool {
	var (
		answer string
		prompt = true
	)
	for prompt {
		fmt.Print("[+] Are you sure?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			return true
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			return false
		} else {
			fmt.Println("[-] Please enter?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
	return false
}
