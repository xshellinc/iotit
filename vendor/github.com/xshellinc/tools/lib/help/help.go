package help

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/hypersleep/easyssh"
	"github.com/mitchellh/go-homedir"
	"github.com/tj/go-spin"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/sudo"
	pb "gopkg.in/cheggaaa/pb.v1"
	"unicode"
)

// Iface represents an entity with interfaceName hardware and ipv4
type Iface struct {
	Name         string
	HardwareAddr string
	Ipv4         string
}

// BackgroundJob contains 2 chans errors and progress indicating if task is in progress
type BackgroundJob struct {
	Progress chan bool
	Err      chan error
}

// Timeouts
const (
	SshExtendedCommandTimeout = 300
	SshCommandTimeout         = 30
)

// Gets homedir based on Os
func UserHomeDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		log.Error(err)
		return ""
	}
	return dir
}

// Returns extract command based on the filename
func GetExtractCommand(file string) string {
	if HasAnySuffixes(file, ".tar.gz", ".tgz", ".tar.bz2", ".tbz", ".tar.xz") {
		return "tar xvf %s -C %s"
	}
	if strings.HasSuffix(file, ".7z") {
		// `/tmp/` is a workaround for the file destination
		return "7z x %s -aos -o%s && 7za l /tmp/" + file + " *.img"
	}
	if strings.HasSuffix(file, "img.xz") {
		file = file[:len(file)-3]
		return "xz -dc %s > %s" + file + " && echo " + file
	}
	if strings.HasSuffix(file, ".zip") {
		return "unzip -o %s -d %s"
	}

	return ""
}

// HasAnySuffixes returns true if file contains any of the supplied suffixes
func HasAnySuffixes(file string, suffix ...string) bool {
	for _, s := range suffix {
		if strings.HasSuffix(file, s) {
			return true
		}
	}

	return false
}

// DeleteHost deletes host from ssh file or any other provided
func DeleteHost(fileName, host string) error {
	result := []string{}
	input, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")
	for _, line := range lines {
		if !strings.Contains(line, host) {
			result = append(result, line)
		}
	}
	output := strings.Join(result, "\n")

	if err = ioutil.WriteFile(fileName, []byte(output), 0644); err != nil {
		return err
	}
	return nil
}

// Separator returns the string separator
func Separator() string {
	return string(filepath.Separator)
}

// Separators returns Os dependent separator
func Separators(os string) string {
	switch os {
	case "windows":
		return "\\"
	default:
		fallthrough
	case "unix":
		return "/"
	}
}

// AddPathSuffix joins path parts based on the OS
func AddPathSuffix(os, path string, suffixes ...string) string {
	s := Separators(os)

	for _, suffix := range suffixes {
		if strings.HasSuffix(path, s) {
			path = path + strings.TrimPrefix(suffix, s)
		} else {
			path = path + s + strings.TrimPrefix(suffix, s)
		}
	}

	return path
}

// ExecStandardStd executes standard command and all in and out channels are redirected to os (i.e. os.Stdout..)
func ExecStandardStd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Executes commands via sudo
func ExecSudo(cb sudo.PasswordCallback, cbData interface{}, script ...string) (string, error) {
	out, eut, err := sudo.Exec(cb, cbData, script...)
	LogCmdErrors(string(out), string(eut), err, script...)
	if err != nil {
		return string(append(out, eut...)), err
	}
	return string(out), err
}

// Executes command
func ExecCmd(cmdName string, cmdArgs []string) (string, error) {
	cmd := exec.Command(cmdName, cmdArgs...)
	cmdOutput := &bytes.Buffer{}
	cmdStdErr := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	cmd.Stderr = cmdStdErr
	err := cmd.Run()

	LogCmdErrors(cmdOutput.String(), cmdStdErr.String(), err, cmd.Args...)
	if err != nil {
		return string(append(cmdOutput.Bytes(), cmdStdErr.Bytes()...)), err
	}
	return cmdOutput.String(), err
}

func LogCmdErrors(out, eut string, err error, args ...string) {
	if err != nil {
		log.Error("Error while executing: `", args, "` error msg: `", eut,
			"` go error:", err.Error())
		log.Error("Output:", out)
	}
}

func CreateFile(path string) {
	// detect if file exists
	var _, err = os.Stat(path)
	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		LogError(err)
		defer file.Close()
	}
}

func CreateDir(path string) error {
	if DirExists(path) {
		return nil
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Error("Error creating dir: ", path, " error msg:", err.Error())
		return err
	}
	return nil
}

func DeleteFile(path string) error {
	log.Debug("Deleting file:", path)
	if !Exists(path) {
		return nil
	}
	err := os.Remove(path)
	if err != nil {
		LogError(err)
		return err
	}
	return nil
}

func WriteFile(path string, content string) {
	// open file using READ & WRITE permission
	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	LogError(err)
	defer file.Close()
	// write some text to file
	_, err = file.WriteString(content)
	LogError(err)
	// save changes
	err = file.Sync()
	LogError(err)
	err = file.Sync()
	LogError(err)
}

func DeleteDir(dir string) error {
	d, err := os.Open(dir)
	log.Debug("DeleteDir func():", "removing dir:", dir)
	if err != nil {
		LogError(err)
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		LogError(err)
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			LogError(err)
			return err
		}
	}
	return nil
}

// DownloadFromUrl downloads target file to destination folder
// create destination dir if does not exist
// download file if does not already exist
// shows progress bar
func DownloadFromUrl(url, destination string) (string, error) {
	var (
		timeout time.Duration = time.Duration(0)
		client  http.Client   = http.Client{Timeout: timeout}
	)
	//tokenize url
	tokens := strings.Split(url, "/")
	//obtain file name
	fileName := tokens[len(tokens)-1]

	// check maybe downloaded file exists and corrupted
	fullFileName := filepath.Join(destination, fileName)
	if _, err := os.Stat(fullFileName); !os.IsNotExist(err) {

		downloadedFileLength, _ := GetFileLength(fullFileName)
		sourceFileLength, _ := GetHTTPFileLength(url)

		if sourceFileLength != downloadedFileLength && sourceFileLength != 0 {
			fmt.Printf("[+] Delete corrupted cached file %s\n", fullFileName)
			DeleteFile(fullFileName)
		}
		// otherwise file has correct length
	}

	fmt.Printf("[+] Downloading %s from %s to %s\n", fileName, url, destination)

	//target file does not exist
	if _, err := os.Stat(fmt.Sprintf("%s/%s", destination, fileName)); os.IsNotExist(err) {
		//create destination dir
		CreateDir(destination)
		//create file
		output, err := os.Create(fmt.Sprintf("%s/%s", destination, fileName))
		if err != nil {
			log.Error("[-] Error creating file ", destination, fileName)
			return "", err
		}
		defer output.Close()
		response, err := client.Get(url)
		if err != nil {
			log.Error("[-] Error while downloading file ", url)
			return "", err
		}
		defer response.Body.Close()
		bar := pb.New64(response.ContentLength)
		bar.ShowBar = false
		bar.SetWidth(100)
		bar.Start()
		prd := bar.NewProxyReader(response.Body)
		totalCount, err := io.Copy(output, prd)
		if err != nil {
			log.Error("error while copying ", err.Error())
			return "", err
		}
		log.Debug("Total number of bytes read: ", totalCount)
	} else {
		fmt.Printf("[+] File exist %s%s\n", destination, fileName)
	}
	fmt.Println("\n[+] Done")
	return fileName, nil
}

func Exists(fullFileName string) bool {
	return DirExists(fullFileName)
}

// DownloadFromUrl downloads target file to destination folder,
// creates destination dir if does not exist
// download file if does not already exist
// shows progress bar
func DownloadFromUrlAsync(url, destination string, readBytesChannel chan int64, errorChan chan error) (string, int64, error) {
	var (
		timeout  time.Duration = time.Duration(0)
		client   http.Client   = http.Client{Timeout: timeout}
		fileName               = "latest"
		length   int64
	)
	response, err := client.Get(url)
	if err != nil {
		log.Error("Error while downloading file from:", url, " error msg:", err.Error())
		return "", 0, err
	}
	contentDisposition := response.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		tokens := strings.Split(url, "/")
		fileName = tokens[len(tokens)-1]
		// log.Error("Error parsing media type ", "error msg:", err.Error())
	} else {
		fileName = params["filename"]
	}
	length = response.ContentLength
	// check maybe downloaded file exists
	fullFileName := filepath.Join(destination, fileName)
	if Exists(fullFileName) {
		downloadedFileLength, _ := GetFileLength(fullFileName)
		sourceFileLength, _ := GetHTTPFileLength(url)
		if sourceFileLength > downloadedFileLength && sourceFileLength != 0 {
			log.Debug("Missing bytes. Resuming download")
			go resumeDownloadAsync(fullFileName, url, readBytesChannel, errorChan)
		} else {
			//report full length
			readBytesChannel <- downloadedFileLength
			//close channel
			close(readBytesChannel)
			close(errorChan)
			return fileName, downloadedFileLength, nil
		}
	} else { //file does not exist, download file
		CreateDir(destination)
		//create file
		output, err := os.Create(fmt.Sprintf("%s%s%s", destination, Separator(), fileName))
		if err != nil {
			log.Error("Error creating file ", destination, fileName)
			return "", 0, err
		}
		// ASYNC part. Download file in background and send read bytes into the readBytesChannel channel
		go func(resp *http.Response, ch chan int64, output *os.File, errorChan chan error) {
			defer response.Body.Close()
			defer close(ch)
			defer output.Close()
			defer close(errorChan)
			prd := NewHttpProxyReader(response.Body, func(n int, err error) {
				ch <- int64(n)
				if err != nil {
					if err.Error() == "EOF" {
						return
					}
					log.WithField("err", err).Error("error occured, sending error down the channel")
					errorChan <- err
					return
				}
			})

			totalCount, err := io.Copy(output, prd)
			if err != nil {
				log.Error("error while copying ", err.Error())
				errorChan <- err
				return
			}
			log.Debug("Total number of bytes read: ", totalCount)

		}(response, readBytesChannel, output, errorChan)
	}
	return fileName, length, nil
}

func resumeDownloadAsync(dst, url string, ch chan int64, errorChan chan error) {
	log.Debug("resuming download to ", dst)
	//resolve redirects to final url
	defer close(ch)
	defer close(errorChan)
	final_url, err := GetFinalUrl(url)
	if err != nil {
		log.Error("http.Get :", err.Error())
	}
	log.Debug("Final resolved URL:", final_url)
	local_length, err := GetFileLength(dst)
	if err != nil {
		log.Error("error getting file length from:", dst, " error msg:", err.Error())
		return
	}
	remote_length, err := GetHTTPFileLength(final_url)
	if err != nil {
		log.Error("error getting remote file length from:", final_url, "error msg:", err.Error())
		return
	}
	log.Debug("Current file size: ", local_length, " remote file length:", remote_length)
	if local_length < remote_length {
		log.Debug("Downloading : ", strconv.FormatInt(remote_length-local_length-1, 10), " bytes")
		//send actual size of the file
		ch <- local_length
		client := &http.Client{}
		req, err := http.NewRequest("GET", final_url, nil)
		if err != nil {
			log.Error("error creating GET request to ", final_url, " error msg:", err.Error())
			return
		}
		range_header := "bytes=" + strconv.FormatInt(local_length, 10) + "-"
		//"-" + strconv.FormatInt(remote_length-1, 10) + "/" + strconv.FormatInt(remote_length, 10)
		req.Header.Add("Range", range_header)
		log.Debug("Adding Range header:", range_header)
		resp, err := client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			log.Error("error making GET request to ", final_url, " error msg:", err.Error())
			return
		}
		log.Debug("Received content length:", resp.ContentLength)
		if resp.StatusCode != http.StatusPartialContent {
			log.Debug("HTTP status code:", resp.StatusCode)
			log.Error("Server does not support Range header, cannot resume download.")
			fmt.Println("\n[-] Server does not support Range header. Cannot resume download. Delete your old file and re-run your command again")
			return
		}

		prd := NewHttpProxyReader(resp.Body, func(n int, err error) {
			ch <- int64(n)
			if err != nil {
				errorChan <- err
			}
		})
		output, err := os.OpenFile(dst, os.O_APPEND|os.O_WRONLY, 0600)
		defer output.Close()
		if err != nil {
			log.Error("error opening file ", "error msg:", err.Error())
		}
		totalCount, err := io.Copy(output, prd)
		if err != nil {
			log.Error("error while copying ", err.Error())
		}
		log.Debug("Total number of bytes read: ", totalCount)
	}
}

func GetFileLength(path string) (int64, error) {
	var file, err = os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return 0, errors.New("File not found " + path)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return 0, errors.New("Can't fetch file info " + path)
	}

	return stat.Size(), nil
}

func GetHTTPFileLength(url string) (int64, error) {
	var (
		timeout time.Duration = time.Duration(0)
		client  http.Client   = http.Client{Timeout: timeout}
	)

	response, err := client.Get(url)
	if err != nil {
		return 0, errors.New("Can't retrieve length of http source " + url)
	}

	return response.ContentLength, nil
}

func GetFinalUrl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	return resp.Request.URL.String(), nil
}

// Downloads From url with retries
func DownloadFromUrlWithAttempts(url, destination string, retries int) (string, error) {
	var (
		err      error
		filename string
	)
	for i := 1; i <= retries; i++ {
		fmt.Printf("[+] Attempting to download. Trying %d out of %d \n", i, retries)
		filename, err = DownloadFromUrl(url, destination)
		if err == nil {
			break
		} else {
			DeleteFile(filepath.Join(destination, filename))
		}
	}
	if err != nil {
		fmt.Printf("[-] Could not download from url:%s \n", url)
		fmt.Printf("[-] Reported error message:%s\n", err.Error())
		return "", err
	}
	return filename, nil

}

// Downloads from the url asynchronously
func DownloadFromUrlWithAttemptsAsync(url, destination string, retries int, wg *sync.WaitGroup) (string, *pb.ProgressBar, error) {
	var (
		err                     error
		filename                string
		destinationFileNameFull string
		length                  int64
		readBytesChannel        = make(chan int64, 10000)
		errorChan               = make(chan error, 1)
	)
	bar := pb.New64(0)
	bar.ShowBar = false
	bar.SetUnits(pb.U_BYTES)
	wg.Add(1)
	go func(ch chan int64, wg *sync.WaitGroup, errorChan chan error, filename *string, url string) {
		// complete task after closing channel
		defer wg.Done()
		for {
			select {
			case n, ok := <-ch:
				if !ok {
					ch = nil
					// log.Debug("Bytes chan is closed")
				}
				bar.Add64(n)
			case err, ok := <-errorChan:
				if !ok {
					errorChan = nil
					// log.Debug("Error chan is closed")
				}
				if err != nil {
					// compare length of files, if length is the same - file is downloaded
					fileLength, _ := GetFileLength(*filename)
					urlFileLength, _ := GetHTTPFileLength(url)
					if fileLength != urlFileLength {
						fmt.Println("[-] Error occured with error message:", err.Error())
						log.Error("Error occured while downloading remote file ", "error msg:", err.Error())
						os.Exit(1)
					}
				}
			}
			if ch == nil && errorChan == nil {
				break
			}

		}

	}(readBytesChannel, wg, errorChan, &destinationFileNameFull, url)

	if filename, length, err = DownloadFromUrlAsync(url, destination, readBytesChannel, errorChan); err != nil {
		fmt.Printf("[-] Could not download from url:%s \n", url)
		fmt.Printf("[-] Reported error message:%s\n", err.Error())
		close(readBytesChannel)
		close(errorChan)
		return "", nil, err
	} else {
		bar.Total = length
	}

	destinationFileNameFull = path.Join(destination, filename)

	return filename, bar, nil

}

func DownloadFile(dst string, url string) (err error) {
	if Exists(dst) {
		downloadedFileLength, _ := GetFileLength(dst)
		sourceFileLength, _ := GetHTTPFileLength(url)
		if downloadedFileLength == sourceFileLength {
			return nil
		}
		DeleteFile(dst)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// GetZipFiles - gets the list of files inside zip archive
func GetZipFiles(src string) ([]*zip.File, error) {
	var files []*zip.File
	r, err := zip.OpenReader(src)
	if err != nil {
		return files, err
	}
	defer r.Close()
	// Iterate through the files in the archive
	for _, f := range r.File {
		files = append(files, f)
	}
	return files, nil
}

// Unzip into the destination folder
func Unzip(src, dest string) error {
	dest = dest + Separator()
	log.WithField("src", src).WithField("dest", dest).Info("Unzipping")
	tokens := strings.Split(src, Separator())
	fileName := tokens[len(tokens)-1]
	// create destination dir with 0777 access rights
	CreateDir(dest)
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}

	var total64 int64
	for _, file := range r.File {
		total64 += int64(file.UncompressedSize64)
	}
	bar := pb.New64(total64).SetUnits(pb.U_BYTES)
	bar.ShowBar = false
	bar.SetMaxWidth(80)
	bar.Prefix(fmt.Sprintf("[+] Unzipping %-15s", fileName))
	for _, f := range r.File {
		bar.Start()
		rc, err := f.Open()
		if err != nil {
			fmt.Println("[-] ", err.Error())
			return err
		}
		defer rc.Close()
		fpath := filepath.Join(dest, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			CreateDir(fpath)
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(fpath, Separator()); lastIndex > -1 {
				fdir = fpath[:lastIndex]
			}
			if err := CreateDir(fdir); err != nil {
				fmt.Println("[-] ", err.Error())
				return err
			}
			dst_f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				fmt.Println("[-] ", err.Error())
				return err
			}
			defer dst_f.Close()
			copied, err := io.Copy(dst_f, rc)
			if err != nil {
				fmt.Println("[-] ", err.Error())
				return err
			}
			bar.Add64(copied)
		}
	}
	bar.Finish()
	time.Sleep(time.Second * 2)
	fmt.Print("[+] Done\n")
	return nil
}

// Appends a string to the provided file
func AppendToFile(s, target string) error {
	fmt.Printf("[+] Appending %s to %s \n", s, target)
	fileHandle, err := os.OpenFile(target, os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		fmt.Printf("[-] Error while appending:%s ", err.Error())
		return err
	}
	defer fileHandle.Close()

	_, err = fileHandle.WriteString(s)
	return err
}

// Writes a string to the provided file
func WriteToFile(s, target string) error {
	fileHandle, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0660)
	if err != nil {
		return err
	}
	defer fileHandle.Close()
	_, err = fileHandle.WriteString(s)

	return err
}

// Get local interfaces with inited ip
func LocalIfaces() ([]Iface, error) {
	var i = make([]Iface, 0)

	ifaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {

			var (
				ip   net.IP
				face Iface
			)

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if !ip.IsLoopback() && ip.To4() != nil && iface.HardwareAddr.String() != "" {
				face.Ipv4 = ip.To4().String()
				face.HardwareAddr = iface.HardwareAddr.String()
				face.Name = iface.Name
				i = append(i, face)
			}
		}
	}

	return i, nil
}

func GetIface() Iface {
	ifaces, _ := LocalIfaces()
	return ifaces[0]
}

/// ------------------- deprecated start =====================

// Stream easy ssh
func StreamEasySsh(ip, user, password, port, key, command string, timeout int) (chan string, chan string, chan bool, error) {
	ssh := &easyssh.MakeConfig{
		User:     user,
		Password: password,
		Port:     port,
		Server:   ip,
		Key:      key,
	}

	return ssh.Stream(fmt.Sprintf("sudo %s", command), timeout)
}

// Scp file
func ScpWPort(src, dst, ip, port, user, password string) error {
	ssh := &easyssh.MakeConfig{
		User:     user,
		Password: password,
		Port:     port,
		Server:   ip,
		Key:      "~/.ssh/id_rsa.pub",
	}

	fileName := FileName(src)
	err := ssh.Scp(src, fileName)
	if err != nil {
		return err
	}

	out, err := GenericRunOverSsh(fmt.Sprintf("mv ~/%s %s", fileName, dst), ip, user, password, port, true, false, SshCommandTimeout)
	if err != nil {
		return errors.New(out)
	}

	return nil
}

// Scp file using 22 port
func Scp(src, dst, ip, user, password string) error {
	return ScpWPort(src, dst, ip, "22", user, password)
}

// Generic command run over ssh, which configures ssh detail and calls RunSshWithTimeout method
func GenericRunOverSsh(command, ip, user, password, port string, sudo bool, verbose bool, timeout int) (string, error) {
	ssh := &easyssh.MakeConfig{
		User:     user,
		Password: password,
		Port:     port,
		Server:   ip,
		Key:      "~/.ssh/id_rsa.pub",
	}

	//if sudo {
	//	command = "sudo " + command
	//}

	if sudo && password != "" {
		command = fmt.Sprintf("sudo -S %s", command)
	}

	if verbose {
		fmt.Printf("[+] Executing %s %s@%s\n", command, user, ip)
	}

	out, eut, t, err := ssh.Run(fmt.Sprintf("echo %s | %s", password, command), timeout)
	if !t {
		fmt.Println("[-] Timeout running command : ", command)
		answ := dialogs.YesNoDialog("Would you like to re-run with extended timeout? ")

		if answ {
			out, eut, t, err = ssh.Run(fmt.Sprintf("echo %s | %s", password, command), SshExtendedCommandTimeout)

			if !t {
				fmt.Println("[-] Timeout running command : ", command)
				return out, errors.New(eut)
			}
		}
	}

	if err != nil {
		fmt.Println("[-] Error running command : ", command, " err msg:", eut)
	}

	return out, err
}

// Run ssh echo password | sudo command with timeout
func RunSudoOverSshTimeout(command, ip, user, password string, timeout int) (string, error) {
	return GenericRunOverSsh(command, ip, user, password, "22", true, false, timeout)
}

// Run ssh echo password | sudo command
func RunSudoOverSsh(command, ip, user, password string, verbose bool) (string, error) {
	return GenericRunOverSsh(command, ip, user, password, "22", true, verbose, SshCommandTimeout)
}

// Run ssh command
func RunOverSsh(command, ip, user, password string) (string, error) {
	return GenericRunOverSsh(command, ip, user, password, "22", false, false, SshCommandTimeout)
}

/// ------------------- deprecated end =====================

// Copy a file
func Copy(src, dst string) error {
	sourcefile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourcefile.Close()
	destfile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destfile.Close()

	if _, err = io.Copy(destfile, sourcefile); err != nil {
		return err
	}
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// Copy a directory recursively
func CopyDir(src, dst string) error {
	// get properties of source dir
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	// create dest dir
	if err = os.MkdirAll(dst, sourceInfo.Mode()); err != nil {
		return err
	}

	directory, _ := os.Open(src)
	defer directory.Close()

	objects, err := directory.Readdir(-1)
	for _, obj := range objects {
		srcp := src + Separator() + obj.Name()
		dstp := dst + Separator() + obj.Name()
		if obj.IsDir() {
			// create sub-directories recursively
			err = CopyDir(srcp, dstp)
			if err != nil {
				fmt.Println(err)
			}
			continue
		}

		// perform copy
		err = Copy(srcp, dstp)
		if err != nil {
			fmt.Println(err)
		}
	}
	return nil
}

// Returns an absolute path of the path
func Abs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Error("Error getting absolute path ", "err msg:", err.Error())
		return ""
	}
	return abs
}

// Returns a default bin path
func GetBinPath() string {
	switch runtime.GOOS {
	case "linux":
		return "/usr/local/bin"
	case "darwin":
		return "/usr/local/bin"
	default:
		return ""
	}
}

// DirExists returns whether the given file or directory exists or not
func DirExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else {
		return !os.IsNotExist(err)
	}
}

// return true if the given file is writable/readable/executable using the given mask by an owner
func FileModeMask(name string, mask os.FileMode) (bool, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return false, err
	}

	return fi.Mode()&mask == mask, nil
}

// Gets an exit code from the error
func CommandExitCode(e error) (int, error) {
	if ee, ok := e.(*exec.ExitError); ok {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			return ws.ExitStatus(), nil
		}
	}

	return 0, errors.New("Wrong error type")
}

// Gets a filename from the path
func FileName(path string) string {
	split := strings.Split(path, Separator())
	name := split[len(split)-1]
	return name
}

func StringToSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

// Shows the message and rotates a spinner while `progress` is true and isn't closed
func WaitAndSpin(message string, progress chan bool) {
	s := spin.New()
	s.Set(spin.Spin1)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	spinEn := true
	ok := false

Loop:
	for {
		select {
		case spinEn, ok = <-progress:
			if !ok {
				fmt.Print("\n")
				break Loop
			}
		case <-ticker.C:
			if spinEn {
				fmt.Printf("\r[+] %s: %s ", message, s.Next())
			}
		}
	}
}

// TODO should replace WaitAndSpin
func NewBackgroundJob() *BackgroundJob {
	return &BackgroundJob{
		Progress: make(chan bool),
		Err:      make(chan error),
	}
}

func (b *BackgroundJob) Error(err error) {
	b.Err <- err
}

func (b *BackgroundJob) Active(active bool) {
	b.Progress <- active
}

func (b *BackgroundJob) Close() {
	close(b.Progress)
}

func WaitJobAndSpin(message string, job *BackgroundJob) (err error) {
	s := spin.New()
	s.Set(spin.Spin1)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	spinEn := true
	ok := false

Loop:
	for {
		select {
		case spinEn, ok = <-job.Progress:
			if !ok {
				fmt.Print("\n")
				break Loop
			}
		case err = <-job.Err:
			fmt.Print("\n")
			break Loop

		case <-ticker.C:
			if spinEn {
				fmt.Printf("\r[+] %s: %s ", message, s.Next())
			}
		}
	}

	return
}

// Logs an error if any
func LogError(err error) {
	if err != nil {
		log.Error(err.Error())
	}
}

// Exits with the code 1 in case of any error
func ExitOnError(err error) {
	if err != nil {
		fmt.Println("[-] Error:", err.Error())
		fmt.Println("[-] Exiting...")
		log.Fatal(err)
	}
}

// Checks connection
func EstablishConn(ip, user, passwd string) bool {
	fmt.Printf("[+] Trying to reach %s@%s\n", user, ip)
	ssh := &easyssh.MakeConfig{
		User:     user,
		Server:   ip,
		Password: passwd,
		Port:     "22",
	}
	resp, eut, t, err := ssh.Run("whoami", SshCommandTimeout)
	if err != nil || !t {
		fmt.Printf("[-] Host is unreachable %s@%s err:%s\n", user, ip, eut)
		return false
	} else {
		fmt.Println("[+] Command `whoami` result: ", strings.Trim(resp, "\n"))
		return true
	}
	return false
}

// HashFileMD5 returns md5 hash for a specified file
func HashFileMD5(filePath string) (string, error) {
	var r string

	file, err := os.Open(filePath)
	if err != nil {
		return r, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return r, err
	}

	return hex.EncodeToString(hash.Sum(nil)[:16]), nil
}

// GetTempDir returns OS specific tmp folder location without trailing slash
func GetTempDir() string {
	return strings.TrimRight(os.TempDir(), Separator())
}

//ValidURL checks if the provided string is a valid URL
func ValidURL(str string) bool {
	u, err := url.Parse(str)
	if err != nil {
		log.Error(err)
		return false
	}
	if u.Scheme == "" || u.Host == "" || u.Path == "" {
		return false
	}
	return true
}

type RuneReader interface {
	ReadRune() (rune, int, error)
	UnreadRune() error
}

type RuneDecimalReader struct {
	RuneReader
}

func (v *RuneDecimalReader) ReadRuneOrDecimal() (val int32, size int, err error) {
	var dec string
	for {
		var s int
		val, s, err = v.ReadRune()

		size += s

		if err != nil || !unicode.IsDigit(val) {
			if dec != "" {
				i, _ := strconv.Atoi(dec)
				val = int32(i)

				if err == nil {
					v.UnreadRune()
					size -= len(string(val))
				}

				err = nil
			}
			return
		}

		dec += string(val)
	}
}

// Compare version strings
// Symbols are compared by their unicode codes, decimal numbers are compared by value,
// so strings "v1" and "v01" are equal and "rev20" is less than "rev120"

func CompareVersions(a, b string) int {
	ar := RuneDecimalReader{RuneReader: strings.NewReader(a)}
	br := RuneDecimalReader{RuneReader: strings.NewReader(b)}

	for {
		av, _, err := ar.ReadRuneOrDecimal()
		if err != nil {
			av = -1
		}

		bv, _, err := br.ReadRuneOrDecimal()
		if err != nil {
			bv = -1
		}

		if av != bv || av == -1 || bv == -1 {
			return int(av - bv)
		}
	}
}

func GetArch() (string, error) {
	a, err := getArch()
	if err != nil {
		return "", nil
	}
	var armPattern = regexp.MustCompile(`^(?i)(armv?[0-9]{1,2})`)
	arch := armPattern.FindString(a)
	if arch != "" {
		return arch, nil
	}

	return a, nil
}
