package workstation

import (
	"fmt"

	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// WorkStation is your computer's Operating System, which should perform specific actions
type WorkStation interface {
	ListRemovableDisk() error
	Unmount() error
	WriteToDisk(img string) (job *help.BackgroundJob, err error)
	Eject() error
	CleanDisk() error
	PrintDisks()
}

// Workstation struct contains parameters such as:
// OS, all available mounts, selected mount to write data and is the mount is writable
type workstation struct {
	Disk     string
	os       string
	writable bool
	mount    *MountInfo
	mounts   []*MountInfo
}

// MountInfo contains mounted disks information
type MountInfo struct {
	deviceName  string
	diskName    string
	diskNameRaw string
	deviceSize  string
}

// NewWorkStation returns workstation depending on the OS
func NewWorkStation(disk string) WorkStation {
	return newWorkstation(disk)
}

// Stringer method
func (m *MountInfo) String() string {
	return fmt.Sprintf("DiskName=%s\n\tdeviceName=%s\n\tdiskNameRaw=%s\n\tdeviceSize=%s",
		m.diskName, m.deviceName, m.diskNameRaw, m.deviceSize)
}

func (w *workstation) GetAvailableDisks(ws WorkStation) []*MountInfo {
	if len(w.mounts) == 0 {
		if err := ws.ListRemovableDisk(); err != nil {
			fmt.Println("[-] SD card not found, please insert an unlocked SD card")
			return []*MountInfo{}
		}
	}
	return w.mounts
}

func (w *workstation) printDisks(ws WorkStation) {
	for _, disk := range w.GetAvailableDisks(ws) {
		fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s"), disk.deviceName, disk.diskName, disk.diskNameRaw)
	}
}
