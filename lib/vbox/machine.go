package vbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	pipeline "github.com/mattn/go-pipeline"
	virtualbox "github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"gopkg.in/cheggaaa/pb.v1"
)

// Stop stops VM
func Stop(name string) error {
	m, err := virtualbox.GetMachine(name)
	if err != nil {
		return err
	}

	if dialogs.YesNoDialog("Would you like to stop the virtual machine?") {
		fmt.Println("[+] Stopping virtual machine")
		if err := m.Poweroff(); err != nil {
			return err
		}
	}

	return nil
}

// CheckMachine checks if any vbox is running with the ability to power-off
// After that imports and runs the vbox image according to the selected device type
func CheckMachine(machine string) error {
	bars := make([]*pb.ProgressBar, 0)
	var wg sync.WaitGroup
	var path = getPath()
	var machinePath = filepath.Join(path, machine, machine+".vbox")

	fmt.Println("[+] Checking virtual machine")
	// checking file location
	if !fileExists(machinePath) {
		repository, err := repo.NewRepositoryVM()

		// checking local repository
		if repository.GetURL() == "" {
			return errors.New("URL is not set for downloading VBox image")
		}

		dst := filepath.Join(repository.Dir(), repository.GetVersion())
		fileName := repository.Name()

		// download virtual machine
		if !fileExists(filepath.Join(dst, fileName)) {
			fmt.Println("[+] Starting virtual machine download")
			var bar1 *pb.ProgressBar
			var err error

			fileName, bar1, err = repo.DownloadAsync(repository, &wg)
			if err != nil {
				return err
			}

			bar1.Prefix(fmt.Sprintf("[+] Download %-15s", fileName))
			if bar1.Total > 0 {
				bars = append(bars, bar1)
			}
			pool, err := pb.StartPool(bars...)
			if err != nil {
				return err
			}
			wg.Wait()
			pool.Stop()

			time.Sleep(time.Second * 2)
		}

		// unzip virtual machine
		err = help.Unzip(filepath.Join(dst, fileName), path)
		if err != nil {
			return err
		}

	}
	if !isActive(machinePath) {
		fmt.Printf("[+] Registering %s\n", machine)
		_, err := help.ExecCmd("VBoxManage",
			[]string{
				"registervm",
				fmt.Sprintf("%s", machinePath),
			})
		if err != nil {
			return err
		}
		fmt.Println("[+] Done")
	}
	fmt.Println("[+] No problem!")
	return nil
}

// Update virtualbox image
func Update() error {
	log.Debug("Virtual Machine Update func()")

	err := CheckVBInstalled()
	help.ExitOnError(err)

	repository, err := repo.NewRepositoryVM()
	if err != nil {
		help.ExitOnError(err)
	}

	if !fileExists(repository.Dir()) {
		fmt.Println("[+] could not find the virtual machine, please execute `iotit`")
	}

	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t WARNNING!!  \t\t\t\t\t\t\t\t\t   *")
	fmt.Println("*\t\t THIS COMMAND WILL INITIALIZE YOUR VBOX SETTINGS!  \t\t\t\t   *")
	fmt.Println("*\t\t IF IT IS OKAY, UPDATE VIRTUAL MACHINE! \t\t\t\t\t   *")
	fmt.Println(strings.Repeat("*", 100))

	if dialogs.YesNoDialog("Would you update virtual machine?") {
		boolean := CheckUpdate()

		if !boolean {
			fmt.Println("[+] Current virtual machine is latest version")
			fmt.Println("[+] Done")
			os.Exit(0)
		}

		var path = getPath()
		var machinePath = filepath.Join(path, VBoxName, VBoxName+".vbox")

		fmt.Println("[+] Unregistering old virtual machine")
		deregister(machinePath)

		// remove old virtual machine
		err = os.RemoveAll(filepath.Join(path, VBoxName))
		if err != nil {
			// rollback virtual machine
			out, err := pipeline.Output(
				[]string{"ls", repository.Dir()},
				[]string{"sort", "-n"},
				[]string{"tail", "-1"},
			)
			if err != nil {
				help.ExitOnError(err)
			}
			currentVersion := strings.Trim(string(out), "\n")

			err = help.Unzip(filepath.Join(repository.Dir(), repository.GetVersion(), currentVersion, VBoxName+".zip"), path)
			if err != nil {
				help.ExitOnError(err)
			}
			_, err = help.ExecCmd("VBoxManage",
				[]string{
					"registervm",
					fmt.Sprintf("%s", machinePath),
				})
			if err != nil {
				help.ExitOnError(err)
			}
			help.ExitOnError(err)
		}

		// download virtual machine
		download(path, repository)

		fmt.Println("[+] Registering new virtual machine")
		_, err = help.ExecCmd("VBoxManage",
			[]string{
				"registervm",
				fmt.Sprintf("%s", machinePath),
			})
		if err != nil {
			help.ExitOnError(err)
		} else {
			fmt.Println("[+] Done")
		}

		conf := filepath.Join(help.UserHomeDir(), ".iotit", "virtualbox", "iotit-vbox.json")
		os.Remove(conf)
	}
	return nil
}

// unregister turns off vm and deregisters it
func deregister(machinePath string) {
	if isActive(VBoxName) {
		m, err := virtualbox.GetMachine(VBoxName)
		if err != nil {
			help.ExitOnError(err)
		}
		if m.State == virtualbox.Running {
			err = m.Poweroff()
			if err != nil {
				help.ExitOnError(err)
			}
		}
		help.ExecCmd("VBoxManage",
			[]string{
				"unregistervm",
				fmt.Sprintf("%s", machinePath),
			})
		fmt.Println("[+] Done")
	}
}

func download(path string, repository repo.Repository) {
	var wg sync.WaitGroup

	bars := make([]*pb.ProgressBar, 0)

	fmt.Println("[+] Starting virtual machine download")
	fileName, bar1, err := repo.DownloadAsync(repository, &wg)
	if err != nil {
		help.ExitOnError(err)
	}
	dst := filepath.Join(repository.Dir(), repository.GetVersion())
	bar1.Prefix(fmt.Sprintf("[+] Download %-15s", fileName))
	if bar1.Total > 0 {
		bars = append(bars, bar1)
	}
	pool, err := pb.StartPool(bars...)
	if err != nil {
		help.ExitOnError(err)
	}
	wg.Wait()
	pool.Stop()
	time.Sleep(time.Second * 2)

	err = help.Unzip(filepath.Join(dst, fileName), path)
	if err != nil {
		help.ExitOnError(err)
	}
}

// CheckVBInstalled checks for virtualbox dependencies
func CheckVBInstalled() error {
	vbm := "VBoxManage"
	if err := exec.Command("which", vbm).Run(); err != nil {
		if runtime.GOOS == "windows" {
			path := `C:\Program Files\Oracle\VirtualBox\`
			if !help.DirExists(path) {
				path = dialogs.GetSingleAnswer("Couldn't find VirtualBox in default location, please specify installation dir manually: ", dialogs.EmptyStringValidator)
				if help.DirExists(path) {
					return fmt.Errorf("Could not find virtualbox, may be you need to install it first")
				}
			}
			exec.Command("setx", "path", `%path%;`+path).Run()
			if err := exec.Command("PowerShell", "-Command", "[Environment]::SetEnvironmentVariable('Path', $env:Path + ';"+path+"', [EnvironmentVariableTarget]::User)").Run(); err != nil {
				log.Error("Error setting PowerShell `path` : ", err.Error())
				return fmt.Errorf(`Couldn't alter your system PATH, please change it manually. E.g. 'C:\Program Files\Oracle\VirtualBox\'`)
			}
			fmt.Println("[+] Added VirtualBox installation directory to system PATH")
		}
		if err := exec.Command("which", vbm).Run(); err != nil {
			log.Error("Error while running `which` : ", err.Error())
			return fmt.Errorf("Could not find virtualbox, please install it first")
		}
	}
	out, _ := exec.Command(vbm, "list", "extpacks").Output()

	match, _ := regexp.MatchString(".*Oracle VM VirtualBox Extension Pack.*", string(out))
	if !match {
		return fmt.Errorf("Could not find virtualbox extension pack, please install virtualbox extension pack, try")
	}
	return nil
}

// CheckUpdate checks for virtualbox image updates
func CheckUpdate() bool {
	log.Debug("Check Update func()")

	err := CheckVBInstalled()
	help.ExitOnError(err)

	var baseDir = filepath.Join(help.UserHomeDir(), ".iotit")
	var vboxDir = filepath.Join(baseDir, "virtualbox")
	var repository repo.Repository
	var currentVersion string
	var comparison = func(s string, width int) (int64, error) {
		strList := strings.Split(s, ".")
		format := fmt.Sprintf("%%s%%0%ds", width)
		v := ""
		for _, value := range strList {
			v = fmt.Sprintf(format, v, value)
		}
		var result int64
		var err error
		result, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return result, nil
	}

	repository, err = repo.NewRepositoryVM()
	help.ExitOnError(err)

	if !fileExists(repository.Dir()) {
		fmt.Println("[+] could not find the virtual machine, lease execute `iotit`")
	}

	newVersion := repository.GetVersion()

	out, err := pipeline.Output(
		[]string{"ls", filepath.Join(vboxDir, "edison")},
		[]string{"sort", "-n"},
		[]string{"tail", "-1"},
	)
	help.ExitOnError(err)
	currentVersion = strings.Trim(string(out), "\n")

	c, err := comparison(currentVersion, 3)
	help.ExitOnError(err)
	n, err := comparison(newVersion, 3)
	help.ExitOnError(err)

	if c < n {
		return true
	}

	return false
}

// StopMachines stops running machines
func StopMachines() error {
	machines, err := virtualbox.ListMachines()
	if err != nil {
		return err
	}
	fmt.Println("[+] Checking running virtual machine")
	for _, m := range machines {
		if m.State == virtualbox.Running {
			var nameStr string
			if m.Description != "" {
				nameStr = m.Description
			} else {
				nameStr = "default"
			}

			if dialogs.YesNoDialog(fmt.Sprintf(dialogs.PrintColored("%s (%s)")+" is running, would like you stop this virtual machine?", m.Name, nameStr)) {
				if err = m.Poweroff(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func getPath() string {
	out, err := virtualbox.SystemProperties()
	if err != nil {
		return err.Error()
	}
	re := regexp.MustCompile(`Default machine folder:(.*)`)
	result := re.FindStringSubmatch(string(out))

	return strings.TrimSpace(result[1])
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func isActive(name string) bool {
	_, err := virtualbox.GetMachine(name)
	return err == nil
}
