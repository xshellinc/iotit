package device

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	virtualbox "github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/device/workstation"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

var ifaces = &Interfaces{
	Address: "192.168.0.254",
	Netmask: "255.255.255.0",
	Gateway: "192.168.0.1",
	Network: "192.168.0.0",
	DNS:     "192.168.0.1",
}

// vbox types
const (
	VBoxTypeDefault = iota
	VBoxTypeNew
	VBoxTypeUser
)

type (
	// Interfaces represents network interfaces used to setup devices
	Interfaces struct {
		Address string
		Gateway string
		Netmask string
		Network string
		DNS     string
	}

	// Wpa supplicant detail used to setup wlan on devices
	wifi struct {
		Name     string
		Password []byte
	}

	// Contains device values and file path's to write these values
	deviceFiles struct {
		locale         string
		localeF        string
		keyboard       string
		keyboardF      string
		wpa            string
		wpaF           string
		interfacesWLAN string
		interfacesEth  string
		interfacesF    string
		resolv         string
		resolvF        string
	}

	// Wrapper on device files collecting files to write
	device struct {
		deviceType string
		*deviceFiles
		files     []string
		writeable bool
	}

	// Mount type
	sd struct {
		*device
	}

	// Mount type
	usb struct {
		*device
	}

	// RaspberryPi mount type
	raspberryPi struct {
		*sd
	}

	// Mount type, contains ip address specific for edison
	edison struct {
		*usb
		ip string
	}

	// NanoPi mount type
	nanoPi struct {
		*sd
	}

	// BeagleBone mount type
	beagleBone struct {
		*sd
	}
)

// SetDevice interface used to setup device's locale, keyboard layout, wifi, static network interfaces
// and upload them into the image
type SetDevice interface {
	SetLocale() error
	SetKeyborad() error
	SetWifi() error
	SetInterfaces(i Interfaces) error
	SelectInterfaces() int
	SetConfig() error
	Upload(*vbox.Config) error
}

// Init starts init process, either by receiving `typeFlag` or providing a user to choose from a list
func Init(typeFlag string) (err error) {
	log.Info("DeviceInit")
	log.Debug("Flag: ", typeFlag)

	devices := [...]string{
		constants.DEVICE_TYPE_RASPBERRY,
		constants.DEVICE_TYPE_EDISON,
		constants.DEVICE_TYPE_NANOPI,
		constants.DEVICE_TYPE_BEAGLEBONE}

	var deviceType string

	if typeFlag != "" {
		if help.StringToSlice(typeFlag, devices[:]) {
			deviceType = typeFlag
		} else {
			fmt.Println("[-]", typeFlag, "device is not supported")
		}
	}

	if deviceType == "" {
		deviceType = devices[dialogs.SelectOneDialog("[?] Select device type: ", devices[:])]
	}

	fmt.Println("[+] flashing", deviceType)

	switch deviceType {
	case constants.DEVICE_TYPE_RASPBERRY:
		return initRasp()
	case constants.DEVICE_TYPE_EDISON:
		return initEdison()
	case constants.DEVICE_TYPE_NANOPI:
		return initNanoPI()
	case constants.DEVICE_TYPE_BEAGLEBONE:
		return initBeagleBone()
	}

	return nil
}

// NewSetDevice creates new device structure for particular device
func NewSetDevice(d string) SetDevice {
	w := &device{d, &deviceFiles{
		constants.LOCALE_LANG + constants.LOCALE + constants.LANG, constants.LOCALE_F,
		constants.KEYMAP, constants.KEYBOAD_F,
		constants.WPA_CONF, constants.WPA_SUPPLICANT,
		constants.INTERFACE_WLAN, constants.INTERFACE_ETH, constants.INTERFACES_F,
		constants.RESOLV, constants.RESOLV_CONF,
	},
		nil, true}

	switch d {
	case constants.DEVICE_TYPE_RASPBERRY:
		return &raspberryPi{&sd{w}}
	case constants.DEVICE_TYPE_EDISON:
		return &edison{&usb{w}, constants.DEFAULT_EDISON_IP}
	case constants.DEVICE_TYPE_NANOPI:
		return &nanoPi{&sd{w}}
	case constants.DEVICE_TYPE_BEAGLEBONE:
		return &beagleBone{&sd{w}}
	default:
		return w
	}
}

// Notifies a user if he wants to change locale and provides with the list of locales
func (d *device) SetLocale() error {
	var (
		prompt  = true
		answer  string
		tmpfile = filepath.Join(constants.TMP_DIR, d.deviceFiles.localeF)
	)

	for prompt {
		fmt.Println("[+] Default language: ", constants.DefaultLocale)
		fmt.Print("[+] Change default language?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
		fmt.Scanln(&answer)

		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			fmt.Print("[+] New locale: ")
			fmt.Scanln(&answer)

			locale, err := selectLocale(constants.GetLocale(answer))
			if err != nil {
				fmt.Println("[-] Error:", err)
				continue
			}

			locale2 := selectLanguagePriority(locale)

			conf := fmt.Sprintf(d.deviceFiles.locale, locale, locale, locale2)
			err = help.WriteToFile(conf, tmpfile)
			if err != nil {
				return err
			}

			d.files = append(d.files, tmpfile)
			prompt = false
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			fmt.Println("[+] Writing default language")
			conf := fmt.Sprintf(d.deviceFiles.locale, constants.DefaultLocale, constants.DefaultLocale, constants.DefaultLocale)
			err := help.WriteToFile(conf, tmpfile)
			if err != nil {
				return err
			}

			d.files = append(d.files, tmpfile)
			prompt = false
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
	return nil
}

// Notifies a user if he wants to change a keyboard layout, user types a layout name
func (d *device) SetKeyborad() error {
	var (
		prompt  = true
		answer  string
		tmpfile = filepath.Join(constants.TMP_DIR, d.deviceFiles.keyboardF)
	)

	for prompt {
		fmt.Println("[+] Default keyboard: ", constants.DefaultKeymap)
		fmt.Print("[+] Change default keyboard?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
		fmt.Scanln(&answer)

		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			fmt.Print("[+] New keyboard: ")
			fmt.Scanln(&answer)
			conf := fmt.Sprintf(d.deviceFiles.keyboard, &answer)
			err := help.WriteToFile(conf, tmpfile)
			if err != nil {
				return err
			}

			d.files = append(d.files, tmpfile)
			prompt = false
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			fmt.Println("[+] Writing default keyboard")
			conf := fmt.Sprintf(d.deviceFiles.keyboard, constants.DefaultKeymap)
			err := help.WriteToFile(conf, tmpfile)
			if err != nil {
				return err
			}

			d.files = append(d.files, tmpfile)
			prompt = false
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
	return nil
}

// Notifies a user, if he wants to set up wifi credentials, user types SSID and pass
func (d *device) SetWifi() error {
	var (
		answer  string
		w       wifi
		prompt  = true
		tmpfile = filepath.Join(constants.TMP_DIR, d.deviceFiles.wpaF)
	)

	for prompt {
		fmt.Print("[+] Would you like to configure your WI-Fi?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			w.Name = strings.TrimSpace(dialogs.WiFiSSIDNameDialog())
			w.Password = []byte(strings.TrimSpace(dialogs.WiFiPassword()))
			prompt = false
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			prompt = false
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}

	if w.Name != "" {
		conf := fmt.Sprintf(d.deviceFiles.wpa, w.Name, w.Password)
		err := help.WriteToFile(conf, tmpfile)
		if err != nil {
			return err
		}
		fmt.Printf("[+] Writing to %s:\n", tmpfile)
		d.files = append(d.files, tmpfile)
	}
	return nil
}

// Notifies a user, if he wants to set a static ip interfaces
func (d *device) SelectInterfaces() int {
	var (
		answer string
		num    = 0
		device = []string{"eth0", "wlan0"}
		prompt = true
	)
	// select network interface
	for prompt {
		fmt.Println("[+] Available network interface: ")
		for i, e := range device {
			fmt.Printf("\t[\x1b[34m%d\x1b[0m] - [\x1b[34m%s\x1b[0m] \n", i, e)
		}
		fmt.Print("[+] Please select a network interface: ")
		fmt.Scanln(&answer)
		n, err := strconv.Atoi(answer)
		if err != nil {
			fmt.Println("[-] Invalid user input")
		} else {
			fmt.Println("[+] Selected:", n)
			//check if outside of range
			if num < 0 || num > len(device)-1 {
				fmt.Printf("[-] Device unavailable with option:%d\n", n)
			} else {
				num = n
				prompt = false
			}
		}
	}
	return num
}

// Users selects either eth0 or wlan0 interface and then he types manually ip addresses for all Interface values
func (d *device) SetInterfaces(i Interfaces) error {
	var (
		answer string
		num    = 0
		s      = 0
		device = []string{"eth0", "wlan0"}
	)

	interfaces := filepath.Join(constants.TMP_DIR, d.deviceFiles.interfacesF)
	resolv := filepath.Join(constants.TMP_DIR, d.deviceFiles.resolvF)

	fmt.Print("[+] Would you like to assign static IP address for your device?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	for {
		// select network interface
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			num = d.SelectInterfaces()

			// assign static ip
			prompt := true
			fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")
			for prompt {
				fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Network:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] Dns:%s\n",
					string(i.Address), string(i.Network), string(i.Gateway), string(i.Netmask), string(i.DNS))
				fmt.Print("[+] Change values?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
				answer = ""
				fmt.Scanln(&answer)
				if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
					setInterfaces(&i)

					switch device[num] {
					case "eth0":
						conf := fmt.Sprintf(d.deviceFiles.interfacesEth, i.Address, i.Netmask, i.Network, i.Gateway, i.DNS)
						err := help.WriteToFile(conf, interfaces)
						if err != nil {
							return err
						}
						d.files = append(d.files, interfaces)

						conf = fmt.Sprintf(d.deviceFiles.resolv, i.DNS)
						err = help.WriteToFile(conf, resolv)
						if err != nil {
							return err
						}
						d.files = append(d.files, resolv)

						fmt.Println("[+]  Ethernet interface configuration was updated")
						s++
					case "wlan0":
						conf := fmt.Sprintf(d.deviceFiles.interfacesWLAN, i.Address, i.Netmask, i.Network, i.Gateway, i.DNS)
						err := help.WriteToFile(conf, interfaces)
						if err != nil {
							return err
						}
						d.files = append(d.files, interfaces)

						conf = fmt.Sprintf(d.deviceFiles.resolv, i.DNS)
						err = help.WriteToFile(conf, resolv)
						if err != nil {
							return err
						}
						d.files = append(d.files, resolv)

						fmt.Println("[+]  wifi interface configuration was updated")
						s++
					}
				} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
					return nil
				} else {
					fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
				}
			}
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			if s > 0 {
				return nil
			}
			return nil
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}

	return nil
}

// @todo make installation from the isaax repo, copy deb packages and install on the first startup
// Notifies user if he wants to install default software package
func (d *device) InitPrograms() error {

	var inp string

	softwareList := [...]string{
		"curl",
		"bluez",
		"iptables",
		"openssh-server",
		"openssh-client",
		"locales",
		"tzdata",
		"sudo",
		"bash",
		"unzip",
		"tar",
		"find",
		"nano",
		"git",
	}

	fmt.Print("[+] Would you like to install basic software for your device?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	for {
		tmpfile := filepath.Join(constants.TMP_DIR, "rc.local.ext")

		fmt.Scan(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			conf := "apt-get update && apt-get install -y " + strings.Join(softwareList[:], " ") + "\nexit 0"
			err := help.WriteToFile(conf, tmpfile)
			if err != nil {
				return err
			}
			d.files = append(d.files, tmpfile)

			return nil

		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			return nil
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}

	return nil
}

// Setup board notification which then triggers setLocale, setWifi, setKeyBoard, InitPrograms, SetInterface methods
func (d *device) SetConfig() error {
	var (
		answer string
		prompt = true
	)

	for prompt {
		fmt.Print("[+] Would you like to config your board?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
		fmt.Scanln(&answer)
		if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") {
			// set locale (host to VM)
			err := d.SetLocale()
			if err != nil {
				return err
			}

			// set keyboard (host to VM)
			err = d.SetKeyborad()
			if err != nil {
				return err
			}

			// wifi config (host to VM)
			err = d.SetWifi()
			if err != nil {
				return err
			}

			// static ip config (host to VM)
			err = d.SetInterfaces(*ifaces)
			if err != nil {
				return err
			}

			err = d.InitPrograms()
			if err != nil {
				return err
			}

			return nil
		} else if strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
			return nil
		} else {
			fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
	return nil
}

// Upload config files to the mounted image on the vbox instance
func (d *device) Upload(vbox *vbox.Config) error {
	if d.writeable == true {
		for _, file := range d.files {
			if _, err := os.Stat(file); !os.IsNotExist(err) {
				fmt.Println("[+] Uploading file : ", file)
				switch help.FileName(file) {
				case "wpa_supplicant.conf":
					err := vbox.SCP(file, filepath.Join(constants.GENERAL_MOUNT_FOLDER, "etc", "wpa_supplicant"))
					os.Remove(file)
					if err != nil {
						return err
					}
				case "interfaces":
					err := vbox.SCP(file, filepath.Join(constants.GENERAL_MOUNT_FOLDER, "etc", "network"))
					os.Remove(file)
					if err != nil {
						return err
					}
				default:
					err := vbox.SCP(file, filepath.Join(constants.GENERAL_MOUNT_FOLDER, "etc"))
					os.Remove(file)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// VirtualBox type option, contains VBOX constant and name used for the list
type vBoxType struct {
	name  string
	vType int
}

// delete host from ssh file or any other provided
func deleteHost(fileName, host string) error {
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
	err = ioutil.WriteFile(fileName, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}

// Creates custom virtualbox specs
func setVbox(v *vbox.Config, conf, template, device string) (*virtualbox.Machine, string, string, error) {
	err := vbox.StopMachines()
	help.ExitOnError(err)

	vboxs := v.Enable(conf, template, device)
	n := selectVboxInit(conf, vboxs)

	switch n {
	case VBoxTypeNew:
		// set up configuration
		v.NameDialog()
		v.DescriptionDialog()
		v.MemoryDialog()
		v.CPUDialog()
		v.USBDialog()
		v.WriteToFile(conf)

		// select virtual machine
		fallthrough
	case VBoxTypeUser:
		// select virtual machine
		vboxs := v.Enable(conf, template, device)
		result := vbox.Select(vboxs)

		// modify virtual machine
		err := result.Modify()
		help.ExitOnError(err)

		// get virtual machine
		m, err := result.Machine()
		return m, result.GetName(), result.GetDescription(), err

	default:
		fallthrough
	case VBoxTypeDefault:
		m, err := virtualbox.GetMachine(template)
		return m, m.Name, "", err
	}
}

// Select option of virtualboxes, default uses default parameters of virtualbox image, others modifies vbox spec
// the name of vbox doesn't change
func selectVboxInit(conf string, v []vbox.Config) int {
	opts := make(map[int]vBoxType)
	n := 0

	opts[n] = vBoxType{"Use default", VBoxTypeDefault}
	n++
	opts[n] = vBoxType{"Create new virtual machine", VBoxTypeNew}
	n++
	opts[n] = vBoxType{"Use your virtual machine", VBoxTypeUser}

	if _, err := os.Stat(conf); os.IsNotExist(err) || v == nil {
		n--
	}

	for {
		for i := 0; i <= n; i++ {
			fmt.Printf("\t[\x1b[34m%d\x1b[0m] - \x1b[34m%s\x1b[0m \n", i, opts[i].name)
		}

		fmt.Print("[+] Please select a number: ")
		var inp int
		_, err := fmt.Scanf("%d", &inp)

		if err != nil || inp < 0 || inp > n {
			fmt.Println("[-] Invalid user input")
			continue
		}

		return opts[inp].vType
	}
}

// Starts a vbox, inits repository, downloads the image into repository, then uploads and unpacks it into the vbox
func vboxDownloadImage(wg *sync.WaitGroup, vBoxTemplate, deviceType string) (*virtualbox.Machine, workstation.WorkStation, *vbox.Config, string) {
	w := workstation.NewWorkStation()
	help.ExitOnError(w.Check("VBoxManage"))

	conf := filepath.Join(help.UserHomeDir(), ".isaax", "virtualbox", "isaax-vbox.json")
	v := vbox.NewConfig(vBoxTemplate, deviceType)
	vm, name, description, err := setVbox(v, conf, vBoxTemplate, deviceType)
	help.ExitOnError(err)

	if vm.State != virtualbox.Running {
		fmt.Printf("[+] Selected virtual machine \n\t[\x1b[34mName\x1b[0m] - \x1b[34m%s\x1b[0m\n\t[\x1b[34mDescription\x1b[0m] - \x1b[34m%s\x1b[0m\n",
			name, description)
		progress := make(chan bool)
		wg.Add(1)
		go func(progress chan bool) {
			defer close(progress)
			defer wg.Done()

			err := vm.Start()
			help.ExitOnError(err)
			time.Sleep(20 * time.Second) // @todo why sleeping here, check workaround
		}(progress)

		help.WaitAndSpin("starting", progress)
		wg.Wait()
	}

	repository, err := repo.NewRepository(deviceType)
	help.ExitOnError(err)
	dst := filepath.Join(repository.Dir(), repository.GetVersion())

	fmt.Println("[+] Starting download ", deviceType)
	zipName, bar, err := repo.DownloadAsync(repository, wg)
	help.ExitOnError(err)

	bar.Prefix(fmt.Sprintf("[+] Download %-15s", zipName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	time.Sleep(time.Second * 2)

	err = deleteHost(filepath.Join((os.Getenv("HOME")), ".ssh", "known_hosts"), "localhost")
	if err != nil {
		log.Error(err)
	}

	// 4. upload edison img
	fmt.Printf("[+] Uploading %s to virtual machine\n", zipName)
	err = v.SCP(filepath.Join(dst, zipName), constants.TMP_DIR)
	help.ExitOnError(err)

	// 5. unzip edison img (in VM)
	fmt.Printf("[+] Extracting %s \n", zipName)
	log.Debug("Extracting an image")
	out, err := v.RunOverSSHExtendedPeriod(fmt.Sprintf("unzip %s -d %s", filepath.Join(constants.TMP_DIR, zipName), constants.TMP_DIR))
	help.ExitOnError(err)

	log.Debug(out)

	str := strings.Split(zipName, ".")

	return vm, w, v, strings.Join(str[:len(str)-1], ".") + ".img"
}
