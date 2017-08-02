package workstation

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

// Linux specific workstation type
type linux struct {
	*workstation
	folder string
}

// Initializes linux workstation with unix type
func newWorkstation(disk string) WorkStation {
	m := new(MountInfo)
	var ms []*MountInfo
	w := &workstation{disk, runtime.GOOS, true, m, ms}
	return &linux{workstation: w, folder: config.MountDir}
}

// Lists available mounts
func (l *linux) ListRemovableDisk() ([]*MountInfo, error) {
	fmt.Println("[+] Listing available disks...")
	regex := regexp.MustCompile(`(sd[a-z])$`)
	regexMmcblk := regexp.MustCompile(`(mmcblk[0-9])$`)
	var (
		devDisks []string
		out      = []*MountInfo{}
	)
	files, _ := ioutil.ReadDir("/dev/")
	for _, f := range files {
		fileName := f.Name()
		if regex.MatchString(fileName) || regexMmcblk.MatchString(fileName) {
			devDisks = append(devDisks, fileName)
		}
	}
	for _, devDisk := range devDisks {
		var p = &MountInfo{}
		diskMap := make(map[string]string)

		r, _ := ioutil.ReadFile("/sys/block/" + devDisk + "/removable")
		removable := strings.Trim(string(r), "\n") == "1"

		sd, _ := ioutil.ReadFile("/sys/block/" + devDisk + "/device/type")
		isSdCard := strings.Trim(string(sd), "\n") == "SD"

		m, _ := ioutil.ReadFile("/sys/block/" + devDisk + "/device/model")
		deviceName := strings.Trim(string(m), "\n")

		// if model is empty, try read from /device/name
		if deviceName == "" {
			n, _ := ioutil.ReadFile("/sys/block/" + devDisk + "/device/name")
			deviceName = strings.Trim(string(n), "\n")
		}

		sizeInSectors, _ := ioutil.ReadFile("/sys/block/" + devDisk + "/size")
		deviceSizeInSectors := strings.Trim(string(sizeInSectors), "\n")
		deviceSizeInSectorsParsed, err := strconv.ParseInt(deviceSizeInSectors, 10, 64)
		if err != nil {
			// unexpected, because there are always integer
			deviceSizeInSectorsParsed = 0
		}

		sectorSize, _ := ioutil.ReadFile("/sys/block/" + devDisk + "/device/erase_size")
		deviceSectorSize := strings.Trim(string(sectorSize), "\n")
		deviceSectorSizeParsed, err := strconv.ParseInt(deviceSectorSize, 10, 64)
		if err != nil {
			// unexpected, because there are always integer
			deviceSectorSizeParsed = 0
		}

		deviceSize := deviceSizeInSectorsParsed * deviceSectorSizeParsed

		diskMap["deviceName"] = deviceName
		diskMap["diskName"] = "/dev/" + devDisk
		diskMap["diskNameRaw"] = "/dev/" + devDisk
		diskMap["deviceSize"] = strconv.FormatInt(deviceSize, 10)

		if removable || isSdCard {
			p.deviceName = diskMap["deviceName"]
			p.diskName = diskMap["diskName"]
			p.diskNameRaw = diskMap["diskNameRaw"]
			p.deviceSize = diskMap["deviceSize"]
			out = append(out, p)
		}
	}

	if !(len(out) > 0) {
		return nil, fmt.Errorf("[-] No mounts found.\n[-] Please insert your SD card and start command again")
	}
	l.workstation.mounts = out
	return out, nil
}

// Unmounts the disk
func (l *linux) Unmount() error {
	if l.workstation.writable != false {
		fmt.Printf("[+] Unmounting disk: %s\n", l.workstation.mount.deviceName)
		stdout, err := help.ExecSudo(sudo.InputMaskedPassword, nil, "umount", l.workstation.mount.diskName)
		if err != nil {
			return fmt.Errorf("Error unmounting disk:%s from %s with error %s, stdout: %s", l.workstation.mount.diskName, l.folder, err.Error(), stdout)
		}
	}
	return nil
}

const diskSelectionTries = 3
const writeAttempts = 3
const cleanTemplate = `n
p
1


w
q
`

// CopyToDisk Notifies user to choose a mount, after that it tries to copy the data
func (l *linux) CopyToDisk(img string) (job *help.BackgroundJob, err error) {
	log.Debug("CopyToDisk")
	_, err = l.ListRemovableDisk()
	if err != nil {
		fmt.Println("[-] SD card is not found, please insert an unlocked SD card")
		return nil, err
	}

	var dev *MountInfo
	if len(l.Disk) == 0 {
		rng := make([]string, len(l.workstation.mounts))
		for i, e := range l.workstation.mounts {
			rng[i] = fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s")+" (%s)", e.deviceName, e.diskName, e.deviceSize)
		}
		num := dialogs.SelectOneDialog("Select disk to format: ", rng)
		dev = l.workstation.mounts[num]
	} else {
		for _, e := range l.workstation.mounts {
			if e.diskName == l.Disk {
				dev = e
				break
			}
		}
		if dev == nil {
			return nil, fmt.Errorf("Disk name not recognised, try to list disks with " + dialogs.PrintColored("disks") + " argument")
		}
	}

	l.workstation.mount = dev
	fmt.Printf("[+] Writing image to %s\n", dev.diskName)
	log.WithField("image", img).WithField("mount", "/media/KERNEL").Debugf("Writing image to %s", dev.diskName)

	if err := l.CleanDisk(dev.diskName); err != nil {
		return nil, err
	}

	job = help.NewBackgroundJob()
	go func() {
		defer job.Close()
		job.Active(true)
		help.ExecCmd("unzip", []string{img, "-d", "/media/KERNEL/"})
		fmt.Println("\r[+] Done writing image to /media/KERNEL")
	}()

	return job, nil
}

// Notifies user to chose a mount, after that it tries to write the data with `diskSelectionTries` number of retries
func (l *linux) WriteToDisk(img string) (job *help.BackgroundJob, err error) {
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
			break
		}

		_, err = l.ListRemovableDisk()
		if err != nil {
			fmt.Println("[-] SD card is not found, please insert an unlocked SD card")
			continue
		}

		var dev *MountInfo
		if len(l.Disk) == 0 {
			fmt.Println("[+] Available mounts: ")

			rng := make([]string, len(l.workstation.mounts))
			for i, e := range l.workstation.mounts {
				rng[i] = fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s"), e.deviceName, e.diskName)
			}
			num := dialogs.SelectOneDialog("Select mount to format: ", rng)
			dev = l.workstation.mounts[num]
		} else {
			for _, e := range l.workstation.mounts {
				if e.diskName == l.Disk {
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
				err = fmt.Errorf("[-] Your card seems locked.\n[-]  Please unlock your SD card and start command again")
			}
		} else {
			l.workstation.mount = dev
			break
		}
	}

	if err != nil {
		return nil, err
	}

	if len(l.Disk) == 0 && !dialogs.YesNoDialog("Are you sure? ") {
		l.workstation.writable = false
		return nil, nil
	}

	fmt.Printf("[+] Writing %s to %s\n", img, l.workstation.mount.diskName)
	fmt.Println("[+] You may need to enter your user password")

	job = help.NewBackgroundJob()

	go func() {
		defer job.Close()

		args := []string{
			"dd",
			fmt.Sprintf("if=%s", img),
			fmt.Sprintf("of=%s", l.workstation.mount.diskName),
			"bs=4M",
		}

		var err error
		for attempt := 0; attempt < writeAttempts; attempt++ {
			if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
				break
			}
			job.Active(true)

			var out, eut []byte
			if out, eut, err = sudo.Exec(sudo.InputMaskedPassword, job.Progress, args...); err != nil {
				help.LogCmdErrors(string(out), string(eut), err, args...)

				job.Active(false)
				fmt.Println("\r[-] Can't write to disk. Please make sure your password is correct")
			} else {
				job.Active(false)
				fmt.Printf("\r[+] Done writing %s to %s \n", img, l.workstation.mount.diskName)
				break
			}
		}

		if err != nil {
			job.Error(err)
		}
	}()

	l.workstation.writable = true
	return job, nil
}

// Ejects the mounted disk
func (l *linux) Eject() error {
	if l.workstation.writable != false {
		fmt.Printf("[+] Eject your sd card :%s\n", l.workstation.mount.diskName)
		eut, err := help.ExecSudo(sudo.InputMaskedPassword, nil, "eject", l.workstation.mount.diskName)
		if err != nil {
			return fmt.Errorf("eject disk failed: %s\n[-] Cause: %s", l.workstation.mount.diskName, eut)
		}
	}
	return nil
}

// CleanDisk does nothing on linux
func (l *linux) CleanDisk(disk string) error {
	log.WithField("disk", disk).Debug("CleanDisk")
	if disk == "" {
		return fmt.Errorf("No disk to format")
	}
	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		job.Active(true)
		help.ExecCmd("umount", []string{disk})

		// fdisk /dev/mmcblk0 < /tmp/fdisk.cmd
		dst := help.GetTempDir() + help.Separator() + "fdisk.cmd"
		help.CreateFile(dst)
		help.WriteFile(dst, cleanTemplate)

		args := []string{
			"fdisk",
			disk,
			"<",
			dst,
		}
		if _, _, err := sudo.Exec(sudo.InputMaskedPassword, job.Progress, args...); err != nil {
			job.Error(err)
		}

		args = []string{
			"mkdosfs",
			"-n",
			"KERNEL",
			disk + "p1",
			"-F",
			"32",
		}
		if _, _, err := sudo.Exec(sudo.InputMaskedPassword, job.Progress, args...); err != nil {
			job.Error(err)
		}
		log.Debug("mkdosfs done")
		help.ExecCmd("mkdir", []string{"/media/KERNEL/"})
		log.Debug("mkdir done")
		help.ExecCmd("mount", []string{disk + "p1", "/media/KERNEL/"})
		log.Debug("mount done")
		l.workstation.mount.deviceName = "/media/KERNEL"
		l.workstation.mount.diskName = disk + "p1"
		l.workstation.writable = true
		job.Active(false)
	}()

	if err := help.WaitJobAndSpin("You need to have "+dialogs.PrintColored("dosfstools")+" package installed. Formatting", job); err != nil {
		return err
	}

	l.workstation.writable = true
	return nil
}

func (l *linux) PrintDisks() {
	l.workstation.printDisks(l)
}
