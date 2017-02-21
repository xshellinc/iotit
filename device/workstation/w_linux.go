package workstation

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

// Linux specific workstation type
type linux struct {
	*workstation
	*unix
}

// Initializes linux workstation with unix type
func newWorkstation() WorkStation {
	m := new(MountInfo)
	var ms []*MountInfo
	w := &workstation{runtime.GOOS, true, m, ms}
	ux := &unix{constants.LINUX_DD, constants.GENERAL_MOUNT_FOLDER, constants.GENERAL_UNMOUNT, constants.GENERAL_EJECT}
	return &linux{workstation: w, unix: ux}
}

// Lists available mounts
func (l *linux) ListRemovableDisk() error {
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
		return fmt.Errorf("[-] No mounts found.\n[-] Please insert your SD card and start command again\n")
	}
	l.workstation.mounts = out
	return nil
}

// Unmounts the disk
func (l *linux) Unmount() error {
	if l.workstation.writable != false {
		fmt.Printf("[+] Unmounting disk:%s\n", l.workstation.mount.deviceName)
		stdout, err := help.ExecSudo(help.InputMaskedPassword, nil, l.unix.unmount, l.workstation.mount.deviceName)
		if err != nil {
			return fmt.Errorf("Error unmounting disk:%s from %s with error %s, stdout: %s", l.workstation.mount.diskName, l.unix.folder, err.Error(), stdout)
		}
	}
	return nil
}

const diskSelectionTries = 3

// Notifies user to chose a mount, after that it tries to write the data with `diskSelectionTries` number of retries
func (l *linux) WriteToDisk(img string) (err error, progress chan bool) {
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("[-] Continue?") {
			break
		}

		err = l.ListRemovableDisk()
		if err != nil {
			fmt.Println("[-] SD card is not found, please insert an unlocked SD card")
			continue
		}

		rng := make([]string, len(l.workstation.mounts))
		for i, e := range l.workstation.mounts {
			rng[i] = fmt.Sprintf("\x1b[34m%s\x1b[0m - \x1b[34m%s\x1b[0m", e.deviceName, e.diskName)
		}
		num := dialogs.SelectOneDialog("[?] Select mount to format: ", rng)

		dev := l.workstation.mounts[num]
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
			l.workstation.mount = dev
			break
		}
	}

	if err != nil {
		return err, nil
	}

	if dialogs.YesNoDialog("[?] Are you sure? ") {
		fmt.Printf("[+] Writing %s to %s\n", img, l.workstation.mount.diskName)
		fmt.Println("[+] You may need to enter user password")

		progress = make(chan bool)

		go func(progress chan bool) {
			defer close(progress)

			for {
				args := []string{
					l.unix.dd,
					fmt.Sprintf("if=%s", img),
					fmt.Sprintf("of=%s", l.workstation.mount.diskName),
					"bs=4M",
				}

				if out, eut, err := sudo.Exec(help.InputMaskedPassword, progress, args...); err != nil {
					help.LogCmdErrors(string(out), string(eut), err, args...)

					progress <- false
					fmt.Println("\r[-] Can't write to disk. Please make sure your password is correct and press Enter to retry")
					fmt.Scanln()
					progress <- true
				} else {
					fmt.Printf("[+] Done writing %s to %s \n", img, l.workstation.mount.diskName)
					break
				}
			}
		}(progress)

		l.workstation.writable = true
		return nil, progress
	} else {
		l.workstation.writable = false
		return nil, progress
	}
}

// Ejects the mounted disk
func (l *linux) Eject() error {
	if l.workstation.writable != false {
		fmt.Printf("[+] Eject your sd card :%s\n", l.workstation.mount.diskName)
		eut, err := help.ExecSudo(help.InputMaskedPassword, nil, l.unix.eject, l.workstation.mount.diskName)
		if err != nil {
			return fmt.Errorf("[-] Error eject disk: %s\n[-] Cause: \n", l.workstation.mount.diskName, eut)
		}
	}
	return nil
}

// Checks virtualbox Dependencies
func (l *linux) Check(pkg string) error {
	return vbox.CheckDeps(pkg)
}
