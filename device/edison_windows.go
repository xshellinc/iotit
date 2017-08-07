package device

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/tools/lib/help"
)

var extractedPath = ""

func (d *edison) Prepare() error {
	fileName := ""
	filePath := ""
	if fn, fp, err := d.flasher.DownloadImage(); err == nil {
		fileName = fn
		filePath = fp
	} else {
		return err
	}

	fmt.Printf("[+] Extracting %s \n", fileName)
	if !strings.HasSuffix(fileName, ".zip") {
		return nil
	}
	extractedPath = help.GetTempDir() + help.Separator() + strings.TrimSuffix(fileName, ".zip") + help.Separator()
	command := "unzip"
	args := []string{
		"-o",
		filePath,
		"-d",
		extractedPath,
	}
	log.WithField("args", args).Debug("Extracting an image...")
	if out, err := exec.Command(command, args...).CombinedOutput(); err != nil {
		log.WithField("out", out).Error(err)
		fmt.Println("[-] Error extracting image!", out)
		return err
	}

	return d.getDFUUtil()
}

func (d *edison) Write() error {
	if extractedPath == "" {
		fmt.Println("[-] Can't use temporary directory")
		return errors.New("Empty extraction path")
	}
	script := extractedPath + help.Separator() + "flashall.bat"
	log.WithField("script", script).Debug("Running flashall... ")
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = extractedPath
	if err := cmd.Run(); err != nil {
		fmt.Println("[-] Can't find Intel Edison board, please try to re-connect it")
		os.Exit(2)
	}
	job := help.NewBackgroundJob()
	go func() {
		defer job.Close()
		time.Sleep(120 * time.Second)
	}()

	help.WaitJobAndSpin("Your Edison board is restarting...", job)
	return nil
}

func (d *edison) getDFUUtil() error {
	dst := extractedPath
	url := "https://cdn.isaax.io/isaax-distro/utilities/dfu-util/dfu-util-0.9-win64.zip"

	if help.Exists(dst + "dfu-util.exe") {
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
	if out, err := exec.Command("unzip", "-j", "-o", dst+"dfu-util-0.9-win64.zip", "-d", dst).CombinedOutput(); err != nil {
		log.Debug(string(out))
		return err
	}
	return nil
}
