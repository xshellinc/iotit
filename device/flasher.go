package device

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// Flasher is an entity for flashing different devices
type Flasher interface {
	PrepareForFlashing() error
	Configure() error
}

// flasher contains virtualbox machine, ssh connection, repository, currently selected device and image name
type flasher struct {
	vbox    *virtualbox.Machine
	conf    *vbox.Config
	devRepo *repo.DeviceMapping

	img    string
	device string
}

// PrepareForFlashing method inits virtualbox, download necessary files from the repo into the vbox
func (d *flasher) PrepareForFlashing() error {
	var name, description string
	var err error
	wg := &sync.WaitGroup{}
	job := help.NewBackgroundJob()

	if err = vbox.CheckVBInstalled(); err != nil {
		return err
	}

	d.conf = vbox.NewConfig(d.device)
	// @todo change name and description
	d.vbox, name, description, err = vbox.SetVbox(d.conf, d.device)
	if err != nil {
		return err
	}

	if d.vbox.State != virtualbox.Running {
		fmt.Printf(`[+] Selected virtual machine
	Name - `+dialogs.PrintColored("%s")+`
	Description - `+dialogs.PrintColored("%s")+"\n", name, description)

		if err := d.vbox.Start(); err != nil {
			return err
		}

		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			defer job.Close()

		Loop:
			for {
				select {
				case <-ticker.C:
					_, eut, err := d.conf.SSH.Run("whoami")
					if err == nil && strings.TrimSpace(eut) == "" {
						fmt.Println("success")
						break Loop
					} else if err != nil {
						log.WithField("error", err).Debug("Connecting SSH")
					}
				case <-time.After(180 * time.Second):
					job.Error(errors.New("Cannot connect to vbox via ssh"))
				}
			}
		}()

		if err := help.WaitJobAndSpin("starting", job); err != nil {
			return err
		}

		time.Sleep(time.Second)
	}

	fmt.Println("[+] Starting download", d.device)

	zipName, bar, err := help.DownloadFromUrlWithAttemptsAsync(d.devRepo.Image.URL, d.devRepo.Dir(), 3, wg)
	if err != nil {
		return err
	}

	bar.Prefix(fmt.Sprintf("[+] Download %-15s", zipName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	time.Sleep(time.Second * 2)

	err = help.DeleteHost(filepath.Join((os.Getenv("HOME")), ".ssh", "known_hosts"), "localhost")
	if err != nil {
		log.Error(err)
	}
	// check if zip is already inside VM
	if _, eut, err := d.conf.SSH.Run("ls " + constants.TMP_DIR + zipName); err != nil || len(strings.TrimSpace(eut)) > 0 {
		fmt.Printf("[+] Uploading %s to virtual machine\n", zipName)
		if err = d.conf.SSH.Scp(help.AddPathSuffix(runtime.GOOS, d.devRepo.Dir(), zipName), constants.TMP_DIR); err != nil {
			return err
		}
	} else {
		log.Debug("Image zip exists inside VM")
	}
	if strings.HasSuffix(zipName, ".zip") {
		if files, err := help.GetZipFiles(help.AddPathSuffix(runtime.GOOS, d.devRepo.Dir(), zipName)); err == nil && len(files) == 1 {
			if _, eut, err := d.conf.SSH.Run("ls " + constants.TMP_DIR + files[0].Name); err == nil && len(strings.TrimSpace(eut)) == 0 {
				log.Debug("Image file already extracted")
				d.img = files[0].Name
			}
		}
	}

	if d.img == "" {
		fmt.Printf("[+] Extracting %s \n", zipName)
		command := fmt.Sprintf(help.GetExtractCommand(zipName), help.AddPathSuffix("unix", constants.TMP_DIR, zipName), constants.TMP_DIR)
		log.Debug("Extracting an image... ", command)
		d.conf.SSH.SetTimer(help.SshExtendedCommandTimeout)

		if out, eut, err := d.conf.SSH.Run(command); err != nil || len(strings.TrimSpace(eut)) > 0 {
			fmt.Println("[-] ", eut)
			return err
		} else {
			log.Debug(out)
			for _, raw := range strings.Split(out, " ") {
				s := strings.TrimSpace(raw)
				if s != "" && strings.HasSuffix(s, ".img") {
					d.img = s
				}
			}
		}
	}

	if d.img == "" {
		return errors.New("Image not found, please check if the repo is valid")
	}

	return nil
}

// Configure is a generic mock method
func (d *flasher) Configure() error {
	fmt.Println("Mock, nothing to configure")
	return nil
}
