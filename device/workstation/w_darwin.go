package workstation

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strings"

	log "github.com/Sirupsen/logrus"
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
	ux := &unix{constants.UnixDD, constants.MountDir, constants.Umount, constants.Eject}

	return &darwin{
		workstation: w,
		unix:        ux,
		unMount:     constants.DarwinUmount,
		diskUtil:    constants.DarwinDiskutil}
}

const diskSelectionTries = 3
const writeAttempts = 3

// Notifies user to chose a mount, after that it tries to write the data with `diskSelectionTries` number of retries
func (d *darwin) WriteToDisk(img string) (job *help.BackgroundJob, err error) {
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
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
		num := dialogs.SelectOneDialog("Select mount to format: ", rng)
		dev := d.workstation.mounts[num]

		if ok, ferr := help.FileModeMask(dev.diskNameRaw, 0200); !ok || ferr != nil {
			if ferr != nil {
				log.Error(ferr)
				break
			} else {
				fmt.Println("[-] Your card seems locked. Please unlock your SD card")
				err = fmt.Errorf("your card seems locked")
			}
		} else {
			d.workstation.mount = dev
			break
		}
	}

	if err != nil {
		return nil, err
	}

	if dialogs.YesNoDialog("Are you sure? ") {
		fmt.Printf("[+] Writing %s to %s\n", img, d.workstation.mount.diskName)
		fmt.Println("[+] You may need to enter your OS X user password")

		job = help.NewBackgroundJob()

		go func() {
			defer job.Close()

			args := []string{
				d.diskUtil,
				d.unMount,
				d.workstation.mount.diskName,
			}

			var err error
			for attempt := 0; attempt < writeAttempts; attempt++ {
				if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
					break
				}
				job.Active(true)

				var eut []byte
				if _, eut, err = sudo.Exec(sudo.InputMaskedPassword, job.Progress, args...); err != nil {

					job.Active(false)
					fmt.Println("\r[-] Can't unmount disk. Please make sure your password is correct and press Enter to retry")
					fmt.Print("\r[-] ", string(eut))
				} else {
					break
				}
			}

			if err != nil {
				job.Error(err)
				return
			}

			args = []string{
				d.unix.dd,
				fmt.Sprintf("if=%s", img),
				fmt.Sprintf("of=%s", d.workstation.mount.diskNameRaw),
				"bs=1048576",
			}

			for attempt := 0; attempt < writeAttempts; attempt++ {
				if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
					break
				}
				job.Active(true)

				var eut []byte
				if _, eut, err = sudo.Exec(sudo.InputMaskedPassword, job.Progress, args...); err != nil {
					job.Active(false)
					fmt.Println("\r[-] Can't write to disk. Please make sure your password is correct")
					fmt.Println("\r[-] ", string(eut))
				} else {
					job.Active(false)
					fmt.Printf("\r[+] Done writing %s to %s \n", img, d.workstation.mount.diskName)
					break
				}
			}

			if err != nil {
				job.Error(err)
			}
		}()

		d.workstation.writable = true
		return job, nil
	}

	d.workstation.writable = false
	return nil, nil
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

		stdout, err := help.ExecCmd(d.diskUtil, []string{"info", "/dev/" + devDisk})
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
		return fmt.Errorf("removable disks not found.\n[-] Please insert your SD card and start command again")
	}

	d.workstation.mounts = out
	return nil
}

// Ejects the mounted disk
func (d *darwin) Eject() error {
	if d.workstation.writable {
		fmt.Printf("[+] Eject your sd card :%s\n", d.workstation.mount.diskName)
		stdout, err := help.ExecSudo(sudo.InputMaskedPassword, nil, d.unix.eject, d.workstation.mount.diskName)

		if err != nil {
			return fmt.Errorf("eject disk failed: %s\n[-] Cause: %s", d.workstation.mount.diskName, stdout)
		}
	}
	return nil
}

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
			return fmt.Errorf("error unmounting disk: %s\n[-] Cause: %s", d.workstation.mount.diskName, stdout)
		}
	}
	return nil
}

// CleanDisk does nothing on macOS
func (d *darwin) CleanDisk() error {
	return nil
}
