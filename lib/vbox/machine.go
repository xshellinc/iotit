package vbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	pipeline "github.com/mattn/go-pipeline"
	virtualbox "github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/lib/constant"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
	"gopkg.in/cheggaaa/pb.v1"
)

var UnknownChoice = errors.New("UNKNOWN CHOICE")

func Stop(name string) error {
	var (
		prompt bool = true
		answer string
	)
	m, err := virtualbox.GetMachine(name)
	if err != nil {
		return err
	}
	for prompt {
		fmt.Print("[+] Would you like to stop the virtual machine?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			fmt.Println("[+] Stopping virtual machine")
			err := m.Poweroff()
			if err != nil {
				return err
			}
			prompt = false
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			fmt.Println("[+] Can not stop virtual machine")
			prompt = false
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
	return nil
}

func CheckMachine(machine, device string) error {
	bars := make([]*pb.ProgressBar, 0)
	var wg sync.WaitGroup
	var repository repo.Repository
	var path = getPath()
	var machine_path = filepath.Join(path, machine, machine+".vbox")

	fmt.Println("[+] Checking virtual machine")
	// checking file location
	if !fileExists(machine_path) {
		if device == constants.DEVICE_TYPE_EDISON {
			repository = repo.VirtualBoxRepositoryEdison()
		} else {
			repository = repo.VirtualBoxRepository()
		}

		// checking local repository
		if repository.GetURL() == "" {
			return errors.New("Url is not set for downloading VBox image")
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
		err := help.Unzip(filepath.Join(dst, fileName), path)
		if err != nil {
			return err
		}

	}
	if !isActive(machine) {
		fmt.Printf("[+] Registering %s\n", machine)
		_, err := help.ExecCmd("VBoxManage",
			[]string{
				"registervm",
				fmt.Sprintf("%s", machine_path),
			})
		if err != nil {
			return err
		}
		fmt.Println("[+] Done")
	}
	fmt.Println("[+] No problem!")
	return nil
}

func VboxUpdate(typeFlag string) (err error) {
	log.Debug("Virtual Machine Update func()")

	switch typeFlag {
	case "sd":
		update(constant.VBOX_TEMPLATE_SD)
	case "edison":
		update(constant.VBOX_TEMPLATE_EDISON)
	}
	return nil
}

func CheckDeps(pkg string) error {
	// @todo replace with help
	err := exec.Command("which", pkg).Run()
	if err != nil {
		log.Error("Error while running `which` : ", err.Error())
		return fmt.Errorf("[-] Could not find virtualbox, please install virtualbox, try")
	} else {
		out, _ := exec.Command("VBoxManage", "list", "extpacks").Output()

		match, _ := regexp.MatchString(".*Oracle VM VirtualBox Extension Pack.*", string(out))
		if !match {
			return fmt.Errorf("[-] Could not find virtualbox extension pack, please install virtualbox extension pack, try")
		}
	}
	return nil
}

func CheckUpdate(typeFlag string) (string, bool) {
	log.Debug("Check Update func()")

	switch typeFlag {
	case "sd":
		b, _ := checkUpdate(constant.VBOX_TEMPLATE_SD)
		if !b {
			fmt.Println("[+] Current virtual machine is latest version")
			fmt.Println("[+] Done")
			return typeFlag, false
		}
	case "edison":
		b, _ := checkUpdate(constant.VBOX_TEMPLATE_EDISON)
		if !b {
			fmt.Println("[+] Current virtual machine is latest version")
			fmt.Println("[+] Done")
			return typeFlag, false
		}
	default:
		fmt.Println("[-] Unknown image ", typeFlag)
		os.Exit(1)
	}

	return typeFlag, true
}

func StopMachines() error {
	var (
		answer string
		prompt bool = true
	)
	machines, err := virtualbox.ListMachines()
	if err != nil {
		return err
	}
	fmt.Println("[+] Checking running virtual machine")
	for _, m := range machines {
		prompt = true
		if m.State == virtualbox.Running {
			fmt.Printf("[+] \x1b[34m%s\x1b[0m is running, would you stop this virtual machine?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):", m.Name)
			for prompt {
				fmt.Scanln(&answer)
				if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
					err = m.Poweroff()
					if err != nil {
						return err
					}
					prompt = false
				} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
					break
				} else {
					fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
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
	return strings.Trim(result[1], " ")

}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func isActive(name string) bool {
	_, err := virtualbox.GetMachine(name)
	return err == nil
}

func checkUpdate(machine string) (bool, error) {
	err := CheckDeps("VBoxManage")
	exit(err)

	var base_dir = filepath.Join(help.UserHomeDir(), ".isaax")
	var vbox_dir = filepath.Join(base_dir, "virtualbox")
	var repository repo.Repository
	var current_version string
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

	if machine == constant.VBOX_TEMPLATE_EDISON {
		repo, err := repo.NewRepositoryVm(constant.VBOX_TEMPLATE_EDISON)
		if err != nil {
			return false, err
		}
		repository = repo
		if !fileExists(repository.Dir()) {
			fmt.Println("[+] could not find the virtual machine, lease execute `isaax device init`")
		}
	} else {
		repo, err := repo.NewRepositoryVm(constant.VBOX_TEMPLATE_SD)
		if err != nil {
			return false, err
		}
		repository = repo
		if !fileExists(repository.Dir()) {
			fmt.Println("[+] could not find the virtual machine, lease execute `isaax device init`")
		}
	}

	new_version := repository.GetVersion()
	if machine == constant.VBOX_TEMPLATE_EDISON {
		out, err := pipeline.Output(
			[]string{"ls", filepath.Join(vbox_dir, "edison")},
			[]string{"sort", "-n"},
			[]string{"tail", "-1"},
		)
		if err != nil {
			return false, err
		}
		current_version = strings.Trim(string(out), "\n")
	} else {
		out, err := pipeline.Output(
			[]string{"ls", filepath.Join(vbox_dir, "sd")},
			[]string{"sort", "-n"},
			[]string{"tail", "-1"},
		)
		if err != nil {
			return false, err
		}
		current_version = strings.Trim(string(out), "\n")
	}

	c, err := comparison(current_version, 3)
	if err != nil {
		return false, err
	}
	n, err := comparison(new_version, 3)
	if err != nil {
		return false, err
	}

	switch {
	case c > n:
		return false, nil
	case c < n:
		return true, nil
	}
	return false, nil
}

func update(machine string) {
	var (
		answer     string
		prompt     bool = true
		wg         sync.WaitGroup
		repository repo.Repository
	)

	bars := make([]*pb.ProgressBar, 0)
	err := CheckDeps("VBoxManage")
	exit(err)

	if machine == constant.VBOX_TEMPLATE_EDISON {
		repo, err := repo.NewRepositoryVm(constant.VBOX_TEMPLATE_EDISON)
		if err != nil {
			exit(err)
		}
		repository = repo
		if !fileExists(repository.Dir()) {
			fmt.Println("[+] could not find the virtual machine, lease execute `isaax device init`")
		}
	} else {
		repo, err := repo.NewRepositoryVm(constant.VBOX_TEMPLATE_SD)
		if err != nil {
			exit(err)
		}
		repository = repo
		if !fileExists(repository.Dir()) {
			fmt.Println("[+] could not find the virtual machine, lease execute `isaax device init`")
		}
	}

	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t WARNNING!!  \t\t\t\t\t\t\t\t\t   *")
	fmt.Println("*\t\t THIS COMMAND WILL INITIALIZE YOUR VBOX SETTINGS!  \t\t\t\t   *")
	fmt.Println("*\t\t IF IT IS OKAY, UPDATE VIRTUAL MACHINE! \t\t\t\t\t   *")
	fmt.Println(strings.Repeat("*", 100))

	fmt.Print("[+] Would you update virtual machine?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	for prompt {
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			boolean, err := checkUpdate(machine)
			if err != nil {
				exit(err)
			}

			if !boolean {
				fmt.Println("[+] Current virtual machine is latest version")
				fmt.Println("[+] Done")
				os.Exit(0)
			}

			var path = getPath()
			var machine_path = filepath.Join(path, machine, machine+".vbox")

			fmt.Println("[+] Unregistering old virtual machine")
			if isActive(machine) {
				m, err := virtualbox.GetMachine(machine)
				if err != nil {
					exit(err)
				}
				if m.State == virtualbox.Running {
					err = m.Poweroff()
					if err != nil {
						exit(err)
					}
				}
				help.ExecCmd("VBoxManage",
					[]string{
						"unregistervm",
						fmt.Sprintf("%s", machine_path),
					})
				fmt.Println("[+] Done")
			}
			// remove old virtual machine
			err = os.RemoveAll(filepath.Join(path, machine))
			if err != nil {
				// rollback virtual machine
				out, err := pipeline.Output(
					[]string{"ls", repository.Dir()},
					[]string{"sort", "-n"},
					[]string{"tail", "-1"},
				)
				if err != nil {
					exit(err)
				}
				current_version := strings.Trim(string(out), "\n")

				err = help.Unzip(filepath.Join(repository.Dir(), repository.GetVersion(), current_version, machine+".zip"), path)
				if err != nil {
					exit(err)
				}
				_, err = help.ExecCmd("VBoxManage",
					[]string{
						"registervm",
						fmt.Sprintf("%s", machine_path),
					})
				if err != nil {
					exit(err)
				}
				exit(err)
			}

			// download virtual machine
			fmt.Println("[+] Starting virtual machine download")
			fileName, bar1, err := repo.DownloadAsync(repository, &wg)
			if err != nil {
				exit(err)
			}
			dst := filepath.Join(repository.Dir(), repository.GetVersion())
			bar1.Prefix(fmt.Sprintf("[+] Download %-15s", fileName))
			if bar1.Total > 0 {
				bars = append(bars, bar1)
			}
			pool, err := pb.StartPool(bars...)
			if err != nil {
				exit(err)
			}
			wg.Wait()
			pool.Stop()
			time.Sleep(time.Second * 2)

			err = help.Unzip(filepath.Join(dst, fileName), path)
			if err != nil {
				exit(err)
			}

			fmt.Println("[+] Registering new virtual machine")
			_, err = help.ExecCmd("VBoxManage",
				[]string{
					"registervm",
					fmt.Sprintf("%s", machine_path),
				})
			if err != nil {
				exit(err)
			} else {
				fmt.Println("[+] Done")
			}

			conf := filepath.Join(help.UserHomeDir(), ".isaax", "virtualbox", "isaax-vbox.json")
			os.Remove(conf)
			return
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			return
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}
