package device

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// Inits vbox, mounts image, copies config files into image, then closes image, copies image into /tmp
// on the host machine, then flashes it onto mounted disk and eject it cleaning up temporary files
func initNanoPI() error {
	wg := &sync.WaitGroup{}

	vm, local, v, img := vboxDownloadImage(wg, lib.VBoxTemplateSD, constants.DEVICE_TYPE_NANOPI)

	// background process
	job := help.NewBackgroundJob()
	wg.Add(1)

	go func() {
		defer job.Close()
		defer wg.Done()

		// 5. attach nanopi img(in VM)
		log.Debug("Attaching an image")
		out, err := v.RunOverSSH(fmt.Sprintf("losetup -f -P %s", filepath.Join(constants.TMP_DIR, img)))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			job.Error(err)
			return
		}
		log.Debug(out)

		// 6. mount loopback device(nanopi img) (in VM)
		log.Debug("Creating tmp folder")
		out, err = v.RunOverSSH(fmt.Sprintf("mkdir -p %s", constants.GENERAL_MOUNT_FOLDER))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			job.Error(err)
			return
		}
		log.Debug(out)

		log.Debug("mounting tmp folder")
		out, err = v.RunOverSSH(fmt.Sprintf("%s -o rw /dev/loop0p2 %s", constants.LINUX_MOUNT, constants.GENERAL_MOUNT_FOLDER))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			job.Error(err)
			return
		}
		log.Debug(out)
	}()

	// 7. setup keyboard, locale, etc...
	config := NewSetDevice(constants.DEVICE_TYPE_NANOPI)
	err := config.SetConfig()
	help.ExitOnError(err)

	// wait background process
	help.ExitOnError(help.WaitJobAndSpin("waiting", job))
	wg.Wait()

	if dialogs.YesNoDialog("Add Google DNS as a secondary NameServer") {
		if _, err := v.RunOverSSH(fmt.Sprintf(AddGoogleNameServerCmd, constants.GENERAL_MOUNT_FOLDER+"etc/dhcp/dhclient.conf")); err != nil {
			return err
		}
	}

	// 8. uploading config
	err = config.Upload(v)
	help.ExitOnError(err)

	// 9. detatch nanopi img(in VM)
	_, err = v.RunOverSSH(fmt.Sprintf("umount %s", constants.GENERAL_MOUNT_FOLDER))
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}
	_, err = v.RunOverSSH("losetup -D")
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}

	// 10. copy nanopi img from VM(VM to host)
	fmt.Println("[+] Starting NanoPI download from virtual machine")
	err = v.Download(img, wg)
	time.Sleep(time.Second * 2)

	// 12. prompt for disk format (in host)
	osImg := filepath.Join(constants.TMP_DIR, img)

	job, err = local.WriteToDisk(osImg)
	help.ExitOnError(err)
	if job != nil {
		help.ExitOnError(help.WaitJobAndSpin("flashing", job))
	}

	err = os.Remove(osImg)
	if err != nil {
		log.Error("[-] Can not remove image: " + err.Error())
	}

	// 13. unmount SD card(in host)
	err = local.Unmount()
	if err != nil {
		log.Error("Error parsing mount option ", "error msg:", err.Error())
	}
	err = local.Eject()
	if err != nil {
		log.Error("Error parsing mount option ", "error msg:", err.Error())
	}

	// 14. Stop VM
	err = vbox.Stop(vm.UUID)
	if err != nil {
		log.Error(err)
	}

	// 15. Info message
	printDoneMessageSd(strings.ToUpper(constants.DEVICE_TYPE_NANOPI), constants.DEFAULT_NANOPI_USERNAME, constants.DEFAULT_NANOPI_PASSWORD)

	return nil
}
