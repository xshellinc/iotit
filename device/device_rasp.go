package device

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

// Inits vbox, mounts image, copies config files into image, then closes image, copies image into /tmp
// on the host machine, then flashes it onto mounted disk and eject it cleaning up temporary files
func initRasp() error {
	wg := &sync.WaitGroup{}

	vm, local, v, img := vboxDownloadImage(wg, lib.VBoxTemplateSD, constants.DEVICE_TYPE_RASPBERRY)

	// background process
	job := help.NewBackgroundJob()
	wg.Add(1)

	go func() {
		defer job.Close()
		defer wg.Done()

		// 5. attach raspbian img(in VM)
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
		// 6. mount loopback device(raspbian img) (in VM)
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
	config := NewSetDevice(constants.DEVICE_TYPE_RASPBERRY)
	err := config.SetConfig()
	help.ExitOnError(err)

	// wait background process
	help.ExitOnError(help.WaitJobAndSpin("waiting", job))
	wg.Wait()

	// 8. uploading config
	err = config.Upload(v)
	help.ExitOnError(err)

	// 9. detatch raspbian img(in VM)
	log.Debug("unmounting tmp folder")
	out, err := v.RunOverSSH(fmt.Sprintf("umount %s", constants.GENERAL_MOUNT_FOLDER))
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}
	log.Debug(out)

	log.Debug("removing losetup")
	out, err = v.RunOverSSH("losetup -D")
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}
	log.Debug(out)

	// 10. copy raspbian img from VM(VM to host)
	fmt.Println("[+] Starting RaspberryPI download from virtual machine")

	err = v.Download(img, wg)
	time.Sleep(time.Second * 2)

	// 13. unmount SD card(in host)
	err = local.Unmount()
	if err != nil {
		log.Error("Error parsing mount option ", "error msg:", err.Error())
	}

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
	printDoneMessageSd("RASPBERRY PI", "pi", "raspberry")

	return nil
}
