package workstation

import (
	"fmt"
	"io"
	"os/exec"
	"syscall"

	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"encoding/csv"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

const cleanTemplate = `
select disk %s
clean
create partition primary
active
assign letter=N
remove letter=N
format fs=fat32 label=New quick
`

type windows struct {
	*workstation
	ddPath string
}

// Initializes windows workstation
func newWorkstation() WorkStation {
	m := new(MountInfo)
	var ms []*MountInfo
	return &windows{&workstation{runtime.GOOS, true, m, ms}, ""}
}

// Lists available mounts
func (w *windows) ListRemovableDisk() error {
	log.Debug("Listing disks...")
	var out = []*MountInfo{}

	// stdout, err := help.ExecCmd("wmic", []string{"diskdrive", "get", "DeviceID,index,InterfaceType,MediaType,Model,Size", "/format:csv"})
	// ugly fix for windows 7 bug where `format:csv` is broken. Also go double escape quoted arguments.
	cmd := exec.Command(`cmd`)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.CmdLine = `cmd /s /c "wmic diskdrive get DeviceID,index,InterfaceType,MediaType,Model,Size /format:"%WINDIR%\System32\wbem\en-US\csv""`
	stdoutb, err := cmd.Output()
	stdout := string(stdoutb)
	log.Debug(stdout)
	if err != nil {
		stdout = ""
	}

	r := csv.NewReader(strings.NewReader(strings.TrimSpace(stdout)))
	r.TrimLeadingSpace = true
	r.Read() //skip the first line
	for {
		if record, err := r.Read(); err == io.EOF {
			break
		} else if err == nil {
			if !strings.Contains(record[4], "Removable") || strings.Contains(record[3], "IDE") {
				continue
			}
			var p = &MountInfo{}
			size := record[6]
			p.deviceSize = size
			sizeInt, _ := strconv.Atoi(size)
			sizeFloat := math.Ceil(float64(sizeInt) / 1024 / 1024 / 1024)
			p.deviceName = record[5] + " [" + strconv.Itoa(int(sizeFloat)) + "GB]"
			p.diskName = `\\?\Device\Harddisk` + record[2] + `\Partition0`
			p.diskNameRaw = record[2]
			out = append(out, p)
		}
	}
	log.WithField("out", out).Debug("got drives")
	if !(len(out) > 0) {
		return fmt.Errorf("[-] No removable disks found, please insert your SD card and try again.\n[-] Please remember to run this tool as an administrator.")
	}
	w.workstation.mounts = out
	return nil
}

// Unmounts the disk
func (w *windows) Unmount() error {
	return nil
}

// Ejects the mounted disk
func (w *windows) Eject() error {
	return nil
}

const diskSelectionTries = 3
const writeAttempts = 5

// Notifies user to choose a mount, after that it tries to write the data with `diskSelectionTries` number of retries
func (w *windows) WriteToDisk(img string) (job *help.BackgroundJob, err error) {
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
			break
		}

		err = w.ListRemovableDisk()
		if err != nil {
			fmt.Println("[-] SD card not found, please insert an unlocked SD card")
			continue
		}

		rng := make([]string, len(w.workstation.mounts))
		for i, e := range w.workstation.mounts {
			rng[i] = fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s"), e.deviceName, e.diskName)
		}
		num := dialogs.SelectOneDialog("Select disk to use: ", rng)

		w.workstation.mount = w.workstation.mounts[num]
		break
	}

	if err != nil {
		return nil, err
	}

	if w.ddPath == "" {
		if err := w.getDDBinary(); err != nil {
			log.Error(err)
			fmt.Println("[-] Error downloading dd binary")
			return nil, err
		}
	}

	if dialogs.YesNoDialog("Are you sure? ") {
		fmt.Printf("[+] Writing %s to %s\n", img, w.workstation.mount.deviceName)

		job = help.NewBackgroundJob()

		go func() {
			defer job.Close()

			var err error
			// may be save state here? So if user canceled operation or error happened he wouldn't need to start the process from the beginning
			for attempt := 0; attempt < writeAttempts; attempt++ {
				if attempt > 0 {
					if !dialogs.YesNoDialog("Retry flashing?") {
						break
					}
				}
				job.Active(true)
				var out []byte
				if out, err = exec.Command(w.ddPath,
					"--filter=removable",
					fmt.Sprintf("if=%s", img),
					fmt.Sprintf("of=%s", w.workstation.mount.diskName),
					"bs=1M").CombinedOutput(); err != nil {
					log.WithField("out", string(out)).Error("Error while executing: `", w.ddPath)
					job.Active(false)
					fmt.Println("\r[-] Can't write to disk.")
				} else {
					sout := string(out)
					job.Active(false)
					log.WithField("out", sout).Debug("dd finished")
					if strings.Contains(sout, "Error ") {
						if strings.Contains(sout, "Access is denied") || strings.Contains(sout, "The device is not ready") {
							fmt.Println("\n[-] Can't write to disk. Please make sure to run this tool as administrator, close all Explorer windows, try reconnecting your disk and finally reboot your computer.\n [-] You can run this tool with `clean` to clean your disk before applying image.")
							if dialogs.YesNoDialog("Or we can try to clean it's partitions right now, should we proceed?") {
								if derr := w.CleanDisk(); derr != nil {
									fmt.Println("[-] Disk cleaning failed:", derr)
									continue
								} else {
									for !dialogs.YesNoDialog("[+] Disk formatted, now please reconnect the device. Type yes once you've done it.") {
									}
								}
							}
							continue
						} else {
							fmt.Println(sout)
							continue
						}
					}
					fmt.Printf("\r[+] Done writing %s to %s \n", img, w.workstation.mount.diskName)
					return
				}
			}

			if err != nil {
				job.Error(err)
			}
			job.Error(fmt.Errorf("Image wasn't flashed"))
		}()
		return job, nil
	}
	return nil, nil
}

func (w *windows) getDDBinary() error {
	dst := help.GetTempDir() + help.Separator()
	url := "https://cdn.isaax.io/isaax-distro/utilities/dd/ddrelease64.zip"

	if help.Exists(dst + "ddrelease64.exe") {
		w.ddPath = dst + "ddrelease64.exe"
		return nil
	}

	wg := &sync.WaitGroup{}
	fileName, bar, err := help.DownloadFromUrlWithAttemptsAsync(url, dst, 5, wg)
	if err != nil {
		return err
	}

	bar.Prefix(fmt.Sprintf("[+] Download %-15s", fileName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	time.Sleep(time.Second)

	log.WithField("dst", dst).Debug("Extracting")
	if out, err := exec.Command("unzip", "-o", dst+"ddrelease64.zip", "-d", dst).CombinedOutput(); err != nil {
		return err
	} else {
		log.Debug(string(out))
	}
	w.ddPath = dst + "ddrelease64.exe"
	return nil
}

// CleanDisk cleans target disk partitions
func (w *windows) CleanDisk() error {
	fmt.Println("[+] Cleaning disk...")
	var last error
	for attempt := 0; attempt < diskSelectionTries; attempt++ {
		if attempt > 0 && !dialogs.YesNoDialog("Continue?") {
			break
		}

		if err := w.ListRemovableDisk(); err != nil {
			fmt.Println("[-] SD card not found, please insert an unlocked SD card")
			last = err
			continue
		}

		rng := make([]string, len(w.workstation.mounts))
		for i, e := range w.workstation.mounts {
			rng[i] = fmt.Sprintf(dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s"), e.deviceName, e.diskName)
		}
		num := dialogs.SelectOneDialog("Select disk to clean: ", rng)

		w.workstation.mount = w.workstation.mounts[num]
		break
	}

	if last != nil {
		return last
	}

	dst := help.GetTempDir() + help.Separator() + "clean_script.txt"
	if dialogs.YesNoDialog("Are you sure you want to clean this disk? ") {
		fmt.Printf("[+] Cleaning disk %s (%s)\n", w.workstation.mount.diskNameRaw, w.workstation.mount.deviceName)
		help.CreateFile(dst)
		help.WriteFile(dst, fmt.Sprintf(cleanTemplate, w.workstation.mount.diskNameRaw))

		if help.Exists(dst) {
			if out, err := exec.Command("diskpart", "/s", dst).CombinedOutput(); err != nil {
				return err
			} else {
				log.Debug(string(out))
				fmt.Println(string(out))
				return nil
			}
		}
	}
	return nil
}
