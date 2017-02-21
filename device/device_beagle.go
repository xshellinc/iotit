package device

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/lib/constant"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

// Inits vbox, mounts image, copies config files into image, then closes image, copies image into /tmp
// on the host machine, then flashes it onto mounted disk and eject it cleaning up temporary files
func initBeagleBone() error {
	wg := &sync.WaitGroup{}

	vm, local, v, img := vboxDownloadImage(wg, constant.VBOX_TEMPLATE_SD, constants.DEVICE_TYPE_BEAGLEBONE)

	// background process
	wg.Add(1)
	progress := make(chan bool)
	go func(progress chan bool) {
		defer close(progress)
		defer wg.Done()

		// 5. mount loopback device(nanopi img) (in VM)
		log.Debug("Creating tmp folder")
		out, err := v.RunOverSsh(fmt.Sprintf("mkdir -p %s", constants.GENERAL_MOUNT_FOLDER))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			help.ExitOnError(err)
		}
		log.Debug(out)

		log.Debug("mounting tmp folder")
		out, err = v.RunOverSsh(fmt.Sprintf("%s -o rw /dev/loop0p1 %s", constants.LINUX_MOUNT, constants.GENERAL_MOUNT_FOLDER))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			help.ExitOnError(err)
		}
		log.Debug(out)

		// 6. disable rename interface name
		out, err = v.RunOverSsh(fmt.Sprintf("ln -sf %s %s/%s", "/dev/null", filepath.Join(constants.GENERAL_MOUNT_FOLDER, "etc", "udev", "rules.d"), "80-net-setup-link.rules"))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			help.ExitOnError(err)
		}
		log.Debug(out)
	}(progress)

	// 7. setup keyboard, locale, etc...
	config := NewSetDevice(constants.DEVICE_TYPE_BEAGLEBONE)
	err := config.SetConfig()
	help.ExitOnError(err)

	// wait background process
	help.WaitAndSpin("waiting", progress)
	wg.Wait()

	// 8. uploading config
	err = config.Upload(v)
	help.ExitOnError(err)

	// 9. detatch beaglebone img(in VM)
	_, err = v.RunOverSsh(fmt.Sprintf("umount %s", constants.GENERAL_MOUNT_FOLDER))
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}
	_, err = v.RunOverSsh("losetup -D")
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}

	// 10. copy beaglebone img from VM(VM to host)
	fmt.Println("[+] Starting Beaglebone download from virtual machine")

	err = v.Download(img, wg)
	time.Sleep(time.Second * 2)

	// 11. remove beaglebone img(in VM)
	//fmt.Println("[+] Removing BeagleBone image from virtual machine")
	//log.Debug("removing image")
	//out, err := v.RunOverSsh(fmt.Sprintf("rm -f %s", filepath.Join(constants.TMP_DIR, zipName)))
	//if err != nil {
	//	log.Error("[-] Error when execute remote command: " + err.Error())
	//}
	//log.Debug(out)
	//
	//out, err = v.RunOverSsh(fmt.Sprintf("rm -f %s", filepath.Join(constants.TMP_DIR, img)))
	//if err != nil {
	//	log.Error("[-] Error when execute remote command: " + err.Error())
	//}
	//log.Debug(out)

	// 12. prompt for disk format (in host)
	osImg := filepath.Join(constants.TMP_DIR, img)

	err, progress = local.WriteToDisk(osImg)
	help.ExitOnError(err)
	help.WaitAndSpin("flashing", progress)

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
	printDoneMessageSd("UBUNTU", "ubuntu", "temppwd")

	return nil
}
