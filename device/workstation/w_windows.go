package workstation

import (
	"fmt"
	"os"
	"os/exec"

	"math"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

// @todo add windows methods

type windows struct {
	*workstation
	ddPath string
}

func newWorkstation() WorkStation {
	m := new(MountInfo)
	var ms []*MountInfo
	return &windows{&workstation{runtime.GOOS, true, m, ms}, "./bin/ddw.exe"}
}

// Lists available mounts
func (w *windows) ListRemovableDisk() error {
	var out = []*MountInfo{}
	rePH := regexp.MustCompile(`PHYSICALDRIVE(\d+)$`)
	reHD := regexp.MustCompile(`Device\\Harddisk(\d+)\\`)
	reSize := regexp.MustCompile(`size is (\d+) bytes`)

	// listDisks := "wmic logicaldisk get caption,description,drivetype,volumename,size"
	//E:       Removable Disk      2          3999825920     FLASH
	// stdout, err := help.ExecCmd("wmic", []string{"logicaldisk", "get", "caption,description,drivetype,volumename,size,deviceid", "/format:csv"})
	stdout, err := help.ExecCmd("wmic", []string{"diskdrive", "list", "brief", "/format:csv"})
	if err != nil {
		stdout = ""
	}
	wmiList := strings.Split(stdout, "\n")
	modelMap := make(map[string]string)
	for _, line := range wmiList {
		result := strings.Split(line, ",")
		log.Debug(line)
		if len(result) < 3 {
			continue
		}
		if len(rePH.FindStringIndex(result[2])) > 0 {
			index := rePH.FindStringSubmatch(result[2])[1]
			modelMap[index] = result[1] //physical id = model name //  \\.\PHYSICALDRIVE0
		}
	}
	log.Debug(modelMap)
	log.Debug("Listing disks...")
	output, err := exec.Command(w.ddPath, "--list", "--filter=removable").CombinedOutput()
	if err != nil {
		log.Error(err)
		fmt.Println("[-] Error getting disk drives list")
	}

	ddListInfo := strings.Split(string(output), "\n")
	diskName := ""
	for _, line := range ddListInfo {
		if diskName == "" && strings.Contains(line, `Device\Harddisk`) {
			diskName = strings.TrimSpace(line)
			continue
		}
		if diskName == "" {
			continue
		}
		log.Info(line, len(reHD.FindStringIndex(diskName)))
		if strings.Contains(line, `size is`) && len(reHD.FindStringIndex(diskName)) > 0 {
			var p = &MountInfo{}
			size := reSize.FindStringSubmatch(line)[1]
			sizeInt, _ := strconv.Atoi(size)
			p.deviceSize = size
			index := reHD.FindStringSubmatch(diskName)[1]
			if deviceName, ok := modelMap[index]; !ok {
				continue
			} else {
				p.deviceName = "[" + strconv.Itoa(int(math.Ceil(float64(sizeInt)/1024/1024/1024))) + "GB] " + deviceName
			}
			p.diskName = diskName
			p.diskNameRaw = diskName
			out = append(out, p)
			diskName = ""
		}
		log.Debug(line)
	}

	if !(len(out) > 0) {
		return fmt.Errorf("[-] No external disks found, remember to run this tool as administrator.\n[-] Please insert your SD card and try again\n")
	}
	w.workstation.mounts = out
	return nil
}

// Unmounts the disk
func (w *windows) Unmount() error {
	if w.workstation.writable != false {
		fmt.Printf("[+] Unmounting disk:%s\n", w.workstation.mount.deviceName)
	}
	return nil
}

const diskSelectionTries = 3
const writeAttempts = 3

// Notifies user to chose a mount, after that it tries to write the data with `diskSelectionTries` number of retries
func (w *windows) WriteToDisk(img string) (job *help.BackgroundJob, err error) {
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
			break
		}

		err = w.ListRemovableDisk()
		if err != nil {
			fmt.Println("[-] SD card is not found, please insert an unlocked SD card")
			continue
		}

		rng := make([]string, len(w.workstation.mounts))
		for i, e := range w.workstation.mounts {
			rng[i] = fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s"), e.deviceName, e.diskName)
		}
		num := dialogs.SelectOneDialog("Select disk to format: ", rng)

		disk := w.workstation.mounts[num]

		if ok, err := help.FileModeMask(disk.diskNameRaw, 0200); !ok || err != nil {
			if err != nil {
				log.Error(err)
				return nil, err

			} else {
				fmt.Println("[-] Your card seems locked. Please unlock your SD card")
				err = fmt.Errorf("[-] Your card seems locked.\n[-]  Please unlock your SD card and start command again\n")
			}
		} else {
			w.workstation.mount = disk
			break
		}
	}

	if err != nil {
		return nil, err
	}

	if dialogs.YesNoDialog("Are you sure? ") {
		fmt.Printf("[+] Writing %s to %s\n", img, w.workstation.mount.diskName)
		fmt.Println("[+] You may need to enter user password")

		job = help.NewBackgroundJob()

		go func() {
			defer job.Close()

			args := []string{
				"dd",
				fmt.Sprintf("if=%s", img),
				fmt.Sprintf("of=%s", w.workstation.mount.diskName),
				"bs=4M",
			}
			fmt.Println(args)
			os.Exit(1)
			var err error
			for attempt := 0; attempt < writeAttempts; attempt++ {
				if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
					break
				}
				job.Active(true)
				// todo: add progress bar?
				var out, eut []byte
				if out, eut, err = sudo.Exec(sudo.InputMaskedPassword, job.Progress, args...); err != nil {
					help.LogCmdErrors(string(out), string(eut), err, args...)

					job.Active(false)
					fmt.Println("\r[-] Can't write to disk. Please make sure your password is correct")
				} else {
					job.Active(false)
					fmt.Printf("\r[+] Done writing %s to %s \n", img, w.workstation.mount.diskName)
					break
				}
			}

			if err != nil {
				job.Error(err)
			}
		}()

		w.workstation.writable = true
		return job, nil
	}

	w.workstation.writable = false
	return nil, nil
}

// Ejects the mounted disk
func (w *windows) Eject() error {
	if w.workstation.writable != false {
		fmt.Printf("[+] Eject your sd card :%s\n", w.workstation.mount.diskName)
	}
	return nil
}
