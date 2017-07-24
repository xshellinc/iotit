package workstation

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

const diskUtil = "diskutil"

// Initializes default workstation
func newWorkstation(disk string) WorkStation {
	m := new(MountInfo)
	var ms []*MountInfo

	return &workstation{disk, runtime.GOOS, true, m, ms}
}

const diskSelectionTries = 3
const writeAttempts = 3

// CopyToDisk Notifies user to choose a mount, after that it tries to copy the data
func (d *workstation) CopyToDisk(img string) (job *help.BackgroundJob, err error) {
	log.Debug("CopyToDisk")
	_, err = d.ListRemovableDisk()
	if err != nil {
		fmt.Println("[-] SD card is not found, please insert an unlocked SD card")
		return nil, err
	}

	var dev *MountInfo
	if len(d.Disk) == 0 {
		rng := make([]string, len(d.mounts))
		for i, e := range d.mounts {
			rng[i] = fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s")+" (%s)", e.deviceName, e.diskName, e.deviceSize)
		}
		num := dialogs.SelectOneDialog("Select disk to format: ", rng)
		dev = d.mounts[num]
	} else {
		for _, e := range d.mounts {
			if e.diskName == d.Disk {
				dev = e
				break
			}
		}
		if dev == nil {
			return nil, fmt.Errorf("Disk name not recognised, try to list disks with " + dialogs.PrintColored("disks") + " argument")
		}
	}

	d.mount = dev
	fmt.Printf("[+] Writing image to %s\n", d.mount.diskName)
	log.WithField("image", img).WithField("mount", "/Volumes/KERNEL").Debugf("Writing image to %s", d.mount.diskName)

	if err := d.CleanDisk(d.mount.diskName); err != nil {
		return nil, err
	}

	job = help.NewBackgroundJob()
	go func() {
		defer job.Close()
		job.Active(true)
		help.ExecCmd("tar", []string{"xf", img, "-C", "/Volumes/KERNEL/"})
		// if err != nil {
		// if !strings.Contains(err.Error(), "Can't create '.'") {
		// job.Active(false)
		// job.Error(err)
		// }
		// }
		fmt.Println("\r[+] Done writing image to /Volumes/KERNEL")
	}()

	return job, nil
}

// Notifies user to choose a mount, after that it tries to write the data with `diskSelectionTries` number of retries
func (d *workstation) WriteToDisk(img string) (job *help.BackgroundJob, err error) {
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
			break
		}

		_, err = d.ListRemovableDisk()
		if err != nil {
			fmt.Println("[-] SD card is not found, please insert an unlocked SD card")
			continue
		}

		var dev *MountInfo
		if len(d.Disk) == 0 {
			fmt.Println("[+] Available mounts: ")

			rng := make([]string, len(d.mounts))
			for i, e := range d.mounts {
				rng[i] = fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s"), e.deviceName, e.diskName)
			}
			num := dialogs.SelectOneDialog("Select mount to format: ", rng)
			dev = d.mounts[num]
		} else {
			for _, e := range d.mounts {
				if e.diskName == d.Disk {
					dev = e
					break
				}
			}
			if dev == nil {
				return nil, fmt.Errorf("Disk name not recognised, try to list disks with " + dialogs.PrintColored("disks") + " argument")
			}
		}

		if ok, ferr := help.FileModeMask(dev.diskNameRaw, 0200); !ok || ferr != nil {
			if ferr != nil {
				log.Error(ferr)
				return nil, ferr
			} else {
				fmt.Println("[-] Your card seems locked. Please unlock your SD card")
				err = fmt.Errorf("[-] Your card seems locked.\n[-]  Please unlock your SD card and start command again\n")
			}
		} else {
			d.mount = dev
			break
		}
	}

	if err != nil {
		return nil, err
	}

	if len(d.Disk) == 0 && !dialogs.YesNoDialog("Are you sure? ") {
		d.writable = false
		return nil, nil
	}

	fmt.Printf("[+] Writing %s to %s\n", img, d.mount.diskName)
	fmt.Println("[+] You may need to enter your OS X user password")

	job = help.NewBackgroundJob()

	go func() {
		defer job.Close()

		args := []string{
			diskUtil,
			"unmountDisk",
			d.mount.diskName,
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
			"dd",
			fmt.Sprintf("if=%s", img),
			fmt.Sprintf("of=%s", d.mount.diskNameRaw),
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
				fmt.Printf("\r[+] Done writing %s to %s \n", img, d.mount.diskName)
				break
			}
		}

		if err != nil {
			job.Error(err)
		}
	}()

	d.writable = true
	return job, nil
}

// Lists available mounts
func (d *workstation) ListRemovableDisk() ([]*MountInfo, error) {
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

		stdout, err := help.ExecCmd(diskUtil, []string{"info", "/dev/" + devDisk})
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
		return nil, fmt.Errorf("removable disks not found.\n[-] Please insert your SD card and start command again")
	}

	d.mounts = out
	return out, nil
}

// Ejects the mounted disk
func (d *workstation) Eject() error {
	if d.writable {
		fmt.Printf("[+] Eject your sd card :%s\n", d.mount.diskName)
		stdout, err := help.ExecSudo(sudo.InputMaskedPassword, nil, "eject", d.mount.diskName)

		if err != nil {
			return fmt.Errorf("eject disk failed: %s\n[-] Cause: %s", d.mount.diskName, stdout)
		}
	}
	return nil
}

// Unmounts the mounted disk
func (d *workstation) Unmount() error {
	if d.writable {
		fmt.Printf("[+] Unmounting your SD card %s\n", d.mount.deviceName)
		stdout, err := help.ExecCmd(diskUtil,
			[]string{
				"unmountDisk",
				d.mount.diskName,
			})
		if err != nil {
			return fmt.Errorf("error unmounting disk: %s\n[-] Cause: %s", d.mount.diskName, stdout)
		}
	}
	return nil
}

// CleanDisk formats disk into single fat32 partition
func (d *workstation) CleanDisk(disk string) error {
	log.Debug("CleanDisk")
	if disk == "" {
		return fmt.Errorf("No disk to format")
	}

	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		job.Active(true)

		args := []string{
			diskUtil,
			"partitionDisk",
			disk,
			"1",
			"mbr",
			"fat32",
			"KERNEL",
			"100%",
		}
		if _, _, err := sudo.Exec(sudo.InputMaskedPassword, job.Progress, args...); err != nil {
			job.Error(err)
		}
		job.Active(false)
	}()

	if err := help.WaitJobAndSpin("Formatting", job); err != nil {
		return err
	}

	d.writable = true
	return nil
}

func (d *workstation) PrintDisks() {
	d.printDisks(d)
}
