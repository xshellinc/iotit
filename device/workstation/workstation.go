package workstation

import "fmt"

// Workstation is your computer's Operating System, which should perform specific actions
type WorkStation interface {
	ListRemovableDisk() error
	Unmount() error
	WriteToDisk(img string) (done chan bool, err error)
	Eject() error
}

// Workstation struct contains parameters such as:
// OS, all available mounts, selected mount to write data and is the mount is writable
type workstation struct {
	os       string
	writable bool
	mount    *MountInfo
	mounts   []*MountInfo
}

// shared type linux/darwin commands
type unix struct {
	dd      string
	folder  string
	unmount string
	eject   string
}

// MountInfo contains mounted disks information
type MountInfo struct {
	deviceName  string
	diskName    string
	diskNameRaw string
	deviceSize  string
}

// Constructor which will return workstation depending on the OS
func NewWorkStation() WorkStation {

	return newWorkstation()
}

// Stringer method
func (m *MountInfo) String() string {
	return fmt.Sprintf("DiskName=%s\n\tdeviceName=%s\n\tdiskNameRaw=%s\n\tdeviceSize=%s",
		m.diskName, m.deviceName, m.diskNameRaw, m.deviceSize)
}
