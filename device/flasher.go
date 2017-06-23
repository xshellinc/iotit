package device

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/device/config"
	"github.com/xshellinc/iotit/repo"
	"github.com/xshellinc/iotit/vbox"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// Flasher is an entity for flashing different devices
type Flasher interface {
	Flash() error
}

// flasher contains virtualbox machine, ssh connection, repository, currently selected device and image name
type flasher struct {
	Quiet   bool
	vbox    *virtualbox.Machine
	conf    *vbox.Config
	devRepo *repo.DeviceMapping

	img    string
	device string
}

// validates given image path and downloads image archive to os tmp folder
func (d *flasher) DownloadImage() (fileName, filePath string, err error) {
	wg := &sync.WaitGroup{}
	if !help.ValidURL(d.devRepo.Image.URL) {
		// local file path given
		filePath = d.devRepo.Image.URL
		fileName = filepath.Base(filePath)
		log.WithField("path", filePath).WithField("name", fileName).Info("Custom image")
		if !help.Exists(filePath) {
			fmt.Println("[-] Image location is neither valid URL nor existing file. Aborting.")
			return "", "", errors.New("Invalid image location")
		}
		fmt.Println("[+] Using local image file for ", d.device)
		return fileName, filePath, err
	}
	// download image over http
	fmt.Println("[+] Starting download", d.device)
	log.WithField("url", d.devRepo.Image.URL).WithField("dir", d.devRepo.Dir()).Debug("download")
	name, bar, err := help.DownloadFromUrlWithAttemptsAsync(d.devRepo.Image.URL, d.devRepo.Dir(), 3, wg)
	if err != nil {
		return "", "", err
	}
	fileName = name
	filePath = filepath.Join(d.devRepo.Dir(), fileName)

	bar.Prefix(fmt.Sprintf("[+] Download %-15s", fileName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	time.Sleep(time.Second * 2)

	return fileName, filePath, err
}

// PrepareForFlashing method inits virtualbox, download necessary files from the repo into the vbox
func (d *flasher) Prepare() error {
	if err := vbox.CheckVBInstalled(); err != nil {
		return err
	}

	d.conf = vbox.NewConfig(d.device)
	// @todo change name and description
	log.Debug("Configuring virtual box")
	var err error
	d.vbox, err = d.conf.GetVbox(d.device, d.Quiet)

	if err != nil {
		return err
	}
	log.WithField("name", d.vbox.Name).Info("Selected profile")

	if d.vbox.State != virtualbox.Running {
		fmt.Printf(`[+] Selected virtual machine
	Name - `+dialogs.PrintColored("%s")+`
	Description - `+dialogs.PrintColored("%s")+"\n", d.vbox.Name, d.vbox.Description)

		if err := d.startVM(); err != nil {
			return err
		}
	}

	help.DeleteHost(filepath.Join(help.UserHomeDir(), ".ssh", "known_hosts"), "localhost")

	fileName := ""
	filePath := ""

	if fn, fp, err := d.DownloadImage(); err == nil {
		fileName = fn
		filePath = fp
	} else {
		return err
	}

	if _, eut, err := d.conf.SSH.Run("ls " + config.TmpDir + fileName); err != nil || len(strings.TrimSpace(eut)) > 0 {
		fmt.Printf("[+] Uploading %s to virtual machine\n", fileName)
		if err := d.conf.SSH.Scp(filePath, config.TmpDir); err != nil {
			return err
		}
	} else {
		log.Debug("Image already exists inside VM")
	}
	log.WithField("path", filePath).WithField("name", fileName).Info("Image")

	if strings.HasSuffix(fileName, ".zip") {
		if files, err := help.GetZipFiles(help.AddPathSuffix(runtime.GOOS, d.devRepo.Dir(), fileName)); err == nil && len(files) == 1 {
			if _, eut, err := d.conf.SSH.Run("ls " + config.TmpDir + files[0].Name); err == nil && len(strings.TrimSpace(eut)) == 0 {
				log.Debug("Image file already extracted")
				d.img = files[0].Name
			}
		}
	}

	if d.img == "" {
		if err := d.extractImage(fileName); err != nil {
			return err
		}
	}

	return nil
}

func (d *flasher) startVM() error {
	job := help.NewBackgroundJob()
	if err := d.vbox.Start(); err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		defer job.Close()
		for {
			select {
			case <-ticker.C:
				_, eut, err := d.conf.SSH.Run("whoami")
				if err == nil && strings.TrimSpace(eut) == "" {
					return
				}
			case <-time.After(180 * time.Second):
				job.Error(errors.New("Cannot connect to vbox via ssh"))
			}
		}
	}()

	if err := help.WaitJobAndSpin("Starting", job); err != nil {
		return err
	}

	time.Sleep(time.Second)
	return nil
}

func (d *flasher) extractImage(fileName string) error {
	fmt.Printf("[+] Extracting %s \n", fileName)
	command := fmt.Sprintf(help.GetExtractCommand(fileName), help.AddPathSuffix("unix", config.TmpDir, fileName), config.TmpDir)
	log.Debug("Extracting an image... ", command)
	d.conf.SSH.SetTimer(help.SshExtendedCommandTimeout)
	out, eut, err := d.conf.SSH.Run(command)
	if err != nil || len(strings.TrimSpace(eut)) > 0 {
		fmt.Println("[-] ", eut)
		return err
	}
	log.Debug(out)
	for _, raw := range strings.Split(out, " ") {
		s := strings.TrimSpace(raw)
		if s != "" && strings.HasSuffix(s, ".img") {
			d.img = s
			return nil
		}
	}
	if out, _, err := d.conf.SSH.Run("ls -1 " + config.TmpDir); err == nil && len(strings.TrimSpace(out)) > 0 {
		log.Debug(out)
		for _, raw := range strings.Split(strings.TrimSpace(out), "\n") {
			s := strings.TrimSpace(raw)
			if s != "" && strings.HasSuffix(s, ".img") {
				d.img = s
				return nil
			}
		}
	}
	return nil
}

// Configure is a generic mock method
func (d *flasher) Configure() error {
	fmt.Println("Mock, nothing to configure")
	return nil
}

// Write is a generic method
func (d *flasher) Write() error {
	fmt.Println("Mock, nothing to write")
	return nil
}

// Flash configures and flashes image
func (d *flasher) Flash() error {
	fmt.Println("Mock, nothing to flash")
	return nil
}

// Done prints out final success message
func (d *flasher) Done() error {
	if err := d.conf.Stop(d.Quiet); err != nil {
		log.Error(err)
	}
	fmt.Println("\t\t ...                      .................    ..                ")
	fmt.Println("\t\t ...                      .................   ....    ...        ")
	fmt.Println("\t\t ...                             ....                 ...        ")
	fmt.Println("\t\t ...          .....              ....                 ...        ")
	fmt.Println("\t\t ...       ...........           ....         ...     .......... ")
	fmt.Println("\t\t ...      ...       ...          ....         ...     ...        ")
	fmt.Println("\t\t ...     ...         ...         ....         ...     ...        ")
	fmt.Println("\t\t ...     ...         ...         ....         ...     ...        ")
	fmt.Println("\t\t ...     ...         ...         ....         ...     ...        ")
	fmt.Println("\t\t ...     ....       ....         ....         ...      ...       ")
	fmt.Println("\t\t ...      .....   .....          ....         ...      ....   .. ")
	fmt.Println("\t\t ...         .......             ....         ...        ....... ")
	fmt.Println("\n\t\t Flashing Complete!")
	fmt.Println("\t\t If you have any questions or suggestions feel free to make an issue at https://github.com/xshellinc/iotit/issues/ or tweet us @isaax_iot")
	return nil
}
