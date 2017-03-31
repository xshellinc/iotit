package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

type DeviceFlasher interface {
	PrepareVbox() error
	Configure() error
}

type deviceFlasher struct {
	vbox     *virtualbox.Machine
	conf     *vbox.Config
	fileName string
	device   string
}

func (d *deviceFlasher) PrepareVbox() error {
	var (
		name, description string
		err               error
	)

	if err = vbox.CheckDeps("VBoxManage"); err != nil {
		return err
	}

	conf := filepath.Join(help.UserHomeDir(), ".iotit", "virtualbox", "iotit-vbox.json")

	d.conf = vbox.NewConfig(d.device)

	// @todo change name and description
	d.vbox, name, description, err = setVbox(d.conf, conf, d.device)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}

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
			time.Sleep(20 * time.Second)
		}(progress)

		help.WaitAndSpin("starting", progress)
		wg.Wait()
	}

	repository, err := repo.NewRepository(d.device)
	if err != nil {
		return err
	}

	dst := filepath.Join(repository.Dir(), repository.GetVersion())

	fmt.Println("[+] Starting download ", d.device)
	zipName, bar, err := repo.DownloadAsync(repository, wg)
	if err != nil {
		return err
	}

	bar.Prefix(fmt.Sprintf("[+] Download %-15s", zipName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	time.Sleep(time.Second * 2)

	err = deleteHost(filepath.Join((os.Getenv("HOME")), ".ssh", "known_hosts"), "localhost")
	if err != nil {
		logrus.Error(err)
	}

	// 4. upload edison img
	fmt.Printf("[+] Uploading %s to virtual machine\n", zipName)
	if err = d.conf.SCP(filepath.Join(dst, zipName), constants.TMP_DIR); err != nil {
		return err
	}

	// 5. unzip edison img (in VM)
	fmt.Printf("[+] Extracting %s \n", zipName)
	logrus.Debug("Extracting an image")
	out, err := d.conf.RunOverSSHExtendedPeriod(fmt.Sprintf("unzip %s -d %s", filepath.Join(constants.TMP_DIR, zipName), constants.TMP_DIR))
	if err != nil {
		return err
	}

	logrus.Debug(out)

	// @todo add unzipped answer
	str := strings.Split(zipName, ".")
	d.fileName = strings.Join(str[:len(str)-1], ".") + ".img"

	return nil
}

func (d *deviceFlasher) Configure() error {
	fmt.Println("standard")
	return nil
}
