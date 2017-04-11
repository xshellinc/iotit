package device

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
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

	if err = vbox.CheckDeps("VBoxManage"); err != nil {
		return err
	}

	d.conf = vbox.NewConfig(d.device)
	// @todo change name and description
	d.vbox, name, description, err = vbox.SetVbox(d.conf, d.device)
	if err != nil {
		return err
	}

	if d.vbox.State != virtualbox.Running {
		fmt.Printf("[+] Selected virtual machine \n\t[\x1b[34mName\x1b[0m] - \x1b[34m%s\x1b[0m\n\t[\x1b[34mDescription\x1b[0m] - \x1b[34m%s\x1b[0m\n",
			name, description)
		progress := make(chan bool)
		wg.Add(1)
		go func(progress chan bool) {
			defer close(progress)
			defer wg.Done()

			err := d.vbox.Start()
			help.ExitOnError(err)
			time.Sleep(45 * time.Second)
		}(progress)

		// @todo replace wait and spin
		help.WaitAndSpin("starting", progress)
		wg.Wait()
	}

	fmt.Println("[+] Starting download ", d.device)

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
		logrus.Error(err)
	}

	fmt.Printf("[+] Uploading %s to virtual machine\n", zipName)
	if err = d.conf.SSH.Scp(help.AddPathSuffix("unix", d.devRepo.Dir(), zipName), constants.TMP_DIR); err != nil {
		return err
	}

	fmt.Printf("[+] Extracting %s \n", zipName)
	logrus.Debug("Extracting an image")
	command := fmt.Sprintf(help.GetExtractCommand(zipName), help.AddPathSuffix("unix", constants.TMP_DIR, zipName), constants.TMP_DIR)
	d.conf.SSH.SetTimer(help.SshExtendedCommandTimeout)
	out, eut, err := d.conf.SSH.Run(command)
	if err != nil || len(strings.TrimSpace(eut)) > 0 {
		fmt.Println("[-] ", eut)
		return err
	}

	logrus.Debug(out)

	for _, raw := range strings.Split(out, " ") {
		s := strings.TrimSpace(raw)
		if s != "" && strings.HasSuffix(s, ".img") {
			d.img = s
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
