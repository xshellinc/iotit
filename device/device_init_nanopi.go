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
)

func initNanoPI() error {
	wg := &sync.WaitGroup{}

	vm, local, v, img := vboxDownloadImage(wg, constant.VBOX_TEMPLATE_SD, constants.DEVICE_TYPE_RASPBERRY)

	// background process
	wg.Add(1)
	progress := make(chan bool)
	go func(progress chan bool) {
		defer close(progress)
		defer wg.Done()

		// 5. attach nanopi img(in VM)
		log.Debug("Attaching an image")
		out, err := v.RunOverSsh(fmt.Sprintf("losetup -f -P %s", filepath.Join(constants.TMP_DIR, img)))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			exitOnError(err)
		}
		log.Debug(out)

		// 6. mount loopback device(nanopi img) (in VM)
		log.Debug("Creating tmp folder")
		out, err = v.RunOverSsh(fmt.Sprintf("mkdir -p %s", constants.GENERAL_MOUNT_FOLDER))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			exitOnError(err)
		}
		log.Debug(out)

		log.Debug("mounting tmp folder")
		out, err = v.RunOverSsh(fmt.Sprintf("%s -o rw /dev/loop0p2 %s", constants.LINUX_MOUNT, constants.GENERAL_MOUNT_FOLDER))
		if err != nil {
			log.Error("[-] Error when execute remote command: " + err.Error())
			exitOnError(err)
		}
		log.Debug(out)
	}(progress)

	// 7. setup keyboard, locale, etc...
	config := NewSetDevice(constants.DEVICE_TYPE_NANOPI)
	err := config.SetConfig()
	exitOnError(err)

	// wait background process
	waitAndSpin("waiting", progress)
	wg.Wait()

	// 8. uploading config
	err = config.Upload(v)
	exitOnError(err)

	// 9. detatch nanopi img(in VM)
	_, err = v.RunOverSsh(fmt.Sprintf("umount %s", constants.GENERAL_MOUNT_FOLDER))
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}
	_, err = v.RunOverSsh("losetup -D")
	if err != nil {
		log.Error("[-] Error when execute remote command: " + err.Error())
	}

	// 10. copy nanopi img from VM(VM to host)
	fmt.Println("[+] Starting NanoPI download from virtual machine")
	err = v.Download(img, wg)
	time.Sleep(time.Second * 2)

	//// 11. remove nanopi img(in VM)
	//fmt.Println("[+] Removing NanoPI image from virtual machine")
	//log.Debug("removing image")
	//out, err := v.RunOverSsh(fmt.Sprintf("rm -f %s", filepath.Join(constants.TMP_DIR, zipName)))
	//if err != nil {
	//	log.Error("[-] Error when execute remote command: " + err.Error())
	//}
	//log.Debug(out)
	//
	//out, err := v.RunOverSsh(fmt.Sprintf("rm -f %s", filepath.Join(constants.TMP_DIR, img)))
	//if err != nil {
	//	log.Error("[-] Error when execute remote command: " + err.Error())
	//}
	//log.Debug(out)

	// 12. prompt for disk format (in host)
	osImg := filepath.Join(constants.TMP_DIR, img)

	err, progress = local.WriteToDisk(osImg)
	exitOnError(err)
	waitAndSpin("flashing", progress)

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
	printDoneMessageSd("NANO PI", "root", "fa")

	return nil
}
