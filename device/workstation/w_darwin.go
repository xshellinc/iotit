package workstation

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

// Darwing specific workstation type, contains unMount and diskUtil strings
// that are used to perform operations on mounted disks
type darwin struct {
	*workstation
	*unix
	unMount  string
	diskUtil string
}

// Initializes darwing workstation with unix and darwin specific program names
func newWorkstation() WorkStation {
	m := new(MountInfo)
	var ms []*MountInfo
	w := &workstation{runtime.GOOS, true, m, ms}
	ux := &unix{constants.LINUX_DD, constants.GENERAL_MOUNT_FOLDER, constants.GENERAL_UNMOUNT, constants.GENERAL_EJECT}

	return &darwin{
		workstation: w,
		unix:        ux,
		unMount:     constants.DARWIN_UNMOUNT_DISK,
		diskUtil:    constants.DARWIN_DISKUTIL}
}

const diskSelectionTries = 3

// Notifies user to chose a mount, after that it tries to write the data with `diskSelectionTries` number of retries
func (d *darwin) WriteToDisk(img string) (err error, progress chan bool) {
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("[-] Continue?") {
			break
		}

		err = d.ListRemovableDisk()
		if err != nil {
			fmt.Println("[-] SD card is not found, please insert an unlocked SD card")
			continue
		}

		fmt.Println("[+] Available mounts: ")

		rng := make([]string, len(d.workstation.mounts))
		for i, e := range d.workstation.mounts {
			rng[i] = fmt.Sprintf("\x1b[34m%s\x1b[0m - \x1b[34m%s\x1b[0m", e.deviceName, e.diskName)
		}
		num := dialogs.SelectOneDialog("[?] Select mount to format: ", rng)
		dev := d.workstation.mounts[num]

		var ok bool
		if ok, err = help.FileModeMask(dev.diskNameRaw, 0200); !ok || err != nil {
			if err != nil {
				log.Error(err)
				return err, nil

			} else {
				fmt.Println("[-] Your card seems locked. Please unlock your SD card")
				err = fmt.Errorf("[-] Your card seems locked.\n[-]  Please unlock your SD card and start command again\n")
			}
		} else {
			d.workstation.mount = dev
			break
		}
	}

	if err != nil {
		return err, nil
	}

	if dialogs.YesNoDialog("[?] Are you sure? ") {
		fmt.Printf("[+] Writing %s to %s\n", img, d.workstation.mount.diskName)
		fmt.Println("[+] You may need to enter your OS X user password")

		//if err = d.Unmount(); err != nil {
		//	//TODO handle error gracefully
		//	log.Error(err.Error())
		//}
		progress = make(chan bool)

		go func(progress chan bool) {
			defer close(progress)

			for {
				args := []string{
					d.diskUtil,
					d.unMount,
					d.workstation.mount.diskName,
				}

				if _, eut, err := sudo.Exec(help.InputMaskedPassword, progress, args...); err != nil {

					progress <- false
					fmt.Println("\r[-] Can't unmount disk. Please make sure your password is correct and press Enter to retry")
					fmt.Print("\r[-] ", string(eut))
					fmt.Scanln()
					progress <- true
				} else {
					break
				}
			}

			for {
				args := []string{
					d.unix.dd,
					fmt.Sprintf("if=%s", img),
					fmt.Sprintf("of=%s", d.workstation.mount.diskNameRaw),
					"bs=1048576",
				}
				if _, eut, err := sudo.Exec(help.InputMaskedPassword, progress, args...); err != nil {

					progress <- false
					fmt.Println("\r[-] Can't write to disk. Please make sure your password is correct and press Enter to retry")
					fmt.Println("\r[-] ", string(eut))
					fmt.Scanln()
					progress <- true
				} else {
					fmt.Printf("[+] Done writing %s to %s \n", img, d.workstation.mount.diskName)
					break
				}
			}
		}(progress)

		d.workstation.writable = true
		return nil, progress
	} else {
		d.workstation.writable = false
		return nil, progress
	}
}

// Lists available mounts
func (d *darwin) ListRemovableDisk() error {
	regex := regexp.MustCompile("^disk([0-9]+)$")
	var (
		devDisks []string
		out      = []*MountInfo{}
	)

	files, _ := ioutil.ReadDir("/dev/")
	for _, f := range files {
		fileName := f.Name()
		if regex.MatchString(fileName) {
			devDisks = append(devDisks, fileName)
		}
	}
	for _, devDisk := range devDisks {
		var p = &MountInfo{}
		diskMap := make(map[string]string)
		removable := true

		stdout, err := help.ExecCmd("diskutil", []string{"info", "/dev/" + devDisk})
		if err != nil {
			stdout = ""
		}
		diskutilInfo := strings.Split(stdout, "\n")
		for _, line := range diskutilInfo {
			if strings.Contains(line, "Protocol") {
				diskProtocol := strings.Trim(strings.Split(line, ":")[1], " ")
				for _, protocol := range []string{"SATA", "ATA", "Disk Image", "PCI", "SAS"} {
					if strings.Contains(diskProtocol, protocol) {
						removable = false
					}
				}
			}
			if strings.Contains(line, "Device Identifier") {
				diskName := strings.Trim(strings.Split(line, ":")[1], " ")
				diskMap["diskName"] = "/dev/" + diskName
				diskMap["diskNameRaw"] = "/dev/r" + diskName
			}

			if strings.Contains(line, "Device / Media Name") {
				deviceName := strings.Trim(strings.Split(line, ":")[1], " ")
				deviceName = strings.Split(deviceName, " Media")[0]
				diskMap["deviceName"] = deviceName
			}

			if strings.Contains(line, "Total Size") {
				deviceSize := strings.Trim(strings.Split(line, ":")[1], " ")
				deviceSize = strings.Split(deviceSize, " (")[0]
				diskMap["deviceSize"] = deviceSize
			}
		}
		if removable {
			p.deviceName = diskMap["deviceName"]
			p.deviceSize = diskMap["deviceSize"]
			p.diskName = diskMap["diskName"]
			p.diskNameRaw = diskMap["diskNameRaw"]
			out = append(out, p)
			log.Debug(diskMap)
		}
	}

	if !(len(out) > 0) {
		return fmt.Errorf("[-] No mounts found.\n[-] Please insert your SD card and start command again\n")
	}

	d.workstation.mounts = out
	return nil
}

// Ejects the mounted disk
func (d *darwin) Eject() error {
	if d.workstation.writable {
		fmt.Printf("[+] Eject your sd card :%s\n", d.workstation.mount.diskName)
		stdout, err := help.ExecSudo(help.InputMaskedPassword, nil, d.unix.eject, d.workstation.mount.diskName)

		if err != nil {
			return fmt.Errorf("[-] Error eject disk: %s\n[-] Cause: %s\n", d.workstation.mount.diskName, stdout)
		}
	}
	return nil
}

// Checks virtualbox Dependencies
func (d *darwin) Check(pkg string) error {
	return vbox.CheckDeps(pkg)
}

//TODO : THIS WILL FAIL(REQUIRES SUDO USER)
// Unmounts the mounted disk
func (d *darwin) Unmount() error {
	if d.workstation.writable {
		fmt.Printf("[+] Unmounting your sd card :%s\n", d.workstation.mount.diskName)
		stdout, err := help.ExecCmd(d.diskUtil,
			[]string{
				d.unMount,
				d.workstation.mount.deviceName,
			})
		if err != nil {
			return fmt.Errorf("[-] Error unmounting disk: %s\n[-] Cause: %s\n", d.workstation.mount.diskName, stdout)
		}
	}
	return nil
}
