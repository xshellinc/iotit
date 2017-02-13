package workstation

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/dialogs"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

type darwin struct {
	*workstation
	*unix
	unMount  string
	diskUtil string
}

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

		var num int
		for {
			var (
				answer string
				err    error
			)
			fmt.Println("[+] Available mounts: ")
			for i, e := range d.workstation.mounts {
				fmt.Printf("\t[\x1b[34m%d\x1b[0m] \x1b[34m%s\x1b[0m - \x1b[34m%s\x1b[0m \n", i, e.deviceName, e.diskName)
			}
			fmt.Print("[?] Select mount to format: ")
			fmt.Scanln(&answer)
			num, err = strconv.Atoi(answer)

			if err != nil {
				fmt.Println("[-] Invalid user input")
				log.Error("Error parsing mount option ", "error msg:", err.Error())
			} else {
				fmt.Println("[+] Selected:", num)
				//check if outside of range
				if num < 0 || num > len(d.workstation.mounts)-1 {
					fmt.Printf("[-] Mount unavailable with option:%d\n", num)
				} else {
					break
				}
			}
		}

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

	if boolean := verify(); boolean == true {
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

				if out, eut, err := sudo.Exec(help.InputPassword, progress, args...); err != nil {
					help.LogCmdErrors(string(out), string(eut), err, args...)

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
				if out, eut, err := sudo.Exec(help.InputPassword, progress, args...); err != nil {
					help.LogCmdErrors(string(out), string(eut), err, args...)

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

		d.workstation.boolean = boolean
		return nil, progress
	} else {
		d.workstation.boolean = boolean
		return nil, progress
	}
}
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
		// @todo is execCmd is sufficient here, or create execCmd with returning stdout, stderr, err?
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

func (d *darwin) Eject() error {
	if d.workstation.boolean != false {
		fmt.Printf("[+] Eject your sd card :%s\n", d.workstation.mount.diskName)
		stdout, err := help.ExecSudo(help.InputPassword, nil, d.unix.eject, d.workstation.mount.diskName)

		if err != nil {
			return fmt.Errorf("[-] Error eject disk: %s\n[-] Cause: %s\n", d.workstation.mount.diskName, stdout)
		}
	}
	return nil
}

func (d *darwin) Check(pkg string) error {
	return vbox.CheckDeps(pkg)
}

//TODO : THIS WILL FAIL(REQUIRES SUDO USER)
func (d *darwin) Unmount() error {
	if d.workstation.boolean != false {
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
