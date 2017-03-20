package device

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
		deviceType = devices[dialogs.SelectOneDialog("Select device type: ", devices[:])]
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
	var conf string
	tmpfile := filepath.Join(constants.TMP_DIR, d.deviceFiles.localeF)

	fmt.Println("[+] Default language: ", constants.DefaultLocale)

	if dialogs.YesNoDialog("Change default language?") {
		inp := dialogs.GetSingleAnswer("New locale: ", dialogs.EmptyStringValidator, dialogs.CreateValidatorFn(constants.ValidateLocale))

		arr, _ := constants.GetLocale(inp)

		var locale string
		if len(arr) == 1 {
			locale = arr[0]
		} else {
			locale = arr[dialogs.SelectOneDialog("Please select a locale from a list", arr)]
		}

		conf = fmt.Sprintf(d.deviceFiles.locale, locale, locale, locale)

	} else {
		conf = fmt.Sprintf(d.deviceFiles.locale, constants.DefaultLocale, constants.DefaultLocale, constants.DefaultLocale)
	}

	fmt.Println("[+] Writing default language")
	if err := help.WriteToFile(conf, tmpfile); err != nil {
		return err
	}

	d.files = append(d.files, tmpfile)

	return nil
}

// Notifies a user if he wants to change a keyboard layout, user types a layout name
func (d *device) SetKeyborad() error {
	var conf string
	tmpfile := filepath.Join(constants.TMP_DIR, d.deviceFiles.keyboardF)

	fmt.Println("[+] Default keyboard: ", constants.DefaultKeymap)

	if dialogs.YesNoDialog("Change default language?") {
		fmt.Print("[+] New keyboard: ")
		var inp string
		fmt.Scanln(&inp)
		conf = fmt.Sprintf(d.deviceFiles.keyboard, &inp)
	} else {
		conf = fmt.Sprintf(d.deviceFiles.keyboard, constants.DefaultKeymap)
	}

	fmt.Println("[+] Writing default keyboard")
	if err := help.WriteToFile(conf, tmpfile); err != nil {
		return err
	}

	d.files = append(d.files, tmpfile)

	return nil
}

// Notifies a user, if he wants to set up wifi credentials, user types SSID and pass
func (d *device) SetWifi() error {
	var w wifi
	tmpfile := filepath.Join(constants.TMP_DIR, d.deviceFiles.wpaF)

	if dialogs.YesNoDialog("Would you like to configure your WI-Fi?") {
		w.Name = dialogs.GetSingleAnswer("[+] WIFI SSID name: ", dialogs.EmptyStringValidator)
		w.Password = []byte(dialogs.WiFiPassword())

		conf := fmt.Sprintf(d.deviceFiles.wpa, w.Name, w.Password)
		if err := help.WriteToFile(conf, tmpfile); err != nil {
			return err
		}
		fmt.Printf("[+] Writing to %s:\n", tmpfile)
		d.files = append(d.files, tmpfile)
	}

	return nil
}

// Users selects either eth0 or wlan0 interface and then he types manually ip addresses for all Interface values
func (d *device) SetInterfaces(i Interfaces) error {
	var conf string
	device := []string{"eth0", "wlan0"}

	interfaces := filepath.Join(constants.TMP_DIR, d.deviceFiles.interfacesF)
	resolv := filepath.Join(constants.TMP_DIR, d.deviceFiles.resolvF)

	if dialogs.YesNoDialog("Would you like to assign static IP address for your device?") {
		fmt.Println("[+] Available network interface: ")
		num := dialogs.SelectOneDialog("Please select a network interface:", device)

		// assign static ip
		fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")

		for {
			fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Network:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
				i.Address, i.Network, i.Gateway, i.Netmask, i.DNS)

			if dialogs.YesNoDialog("Change values?") {
				setInterfaces(&i)

				switch device[num] {
				case "eth0":
					conf = fmt.Sprintf(d.deviceFiles.interfacesEth, i.Address, i.Netmask, i.Network, i.Gateway, i.DNS)
					fmt.Println("[+]  Ethernet interface configuration was updated")
				case "wlan0":
					conf = fmt.Sprintf(d.deviceFiles.interfacesWLAN, i.Address, i.Netmask, i.Network, i.Gateway, i.DNS)
					fmt.Println("[+]  wifi interface configuration was updated")
				}

				if err := help.WriteToFile(conf, interfaces); err != nil {
					return err
				}
				d.files = append(d.files, interfaces)

				conf = fmt.Sprintf(d.deviceFiles.resolv, i.DNS)
				if err := help.WriteToFile(conf, resolv); err != nil {
					return err
				}
				d.files = append(d.files, resolv)
			} else {
				break
			}
		}
	}

	return nil
}

// @todo make installation from the isaax repo, copy deb packages and install on the first startup
// Notifies user if he wants to install default software package
func (d *device) InitPrograms() error {
	tmpfile := filepath.Join(constants.TMP_DIR, "rc.local.ext")

	softwareList := []string{
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

	fmt.Print("  [+]")
	fmt.Print(strings.Join(softwareList, "\n  [+]"))
	fmt.Println()
	if dialogs.YesNoDialog("Would you like to install basic software for your device?") {
		conf := "apt-get update && apt-get install -y " + strings.Join(softwareList[:], " ") + "\nexit 0"
		if err := help.WriteToFile(conf, tmpfile); err != nil {
			return err
		}
		d.files = append(d.files, tmpfile)
	}

	return nil
}

// Setup board notification which then triggers setLocale, setWifi, setKeyBoard, InitPrograms, SetInterface methods
func (d *device) SetConfig() error {

	if dialogs.YesNoDialog("Would you like to config your board?") {
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

	if err = ioutil.WriteFile(fileName, []byte(output), 0644); err != nil {
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
	opts := []string{
		"Use default",
		"Create new virtual machine",
		"Use your virtual machine",
	}
	optTypes := []int{
		VBoxTypeDefault,
		VBoxTypeNew,
		VBoxTypeUser,
	}
	n := len(opts)

	if _, err := os.Stat(conf); os.IsNotExist(err) || v == nil {
		n--
	}

	return optTypes[dialogs.SelectOneDialog("Please select an option: ", opts[:n])]
}

// Starts a vbox, inits repository, downloads the image into repository, then uploads and unpacks it into the vbox
func vboxDownloadImage(wg *sync.WaitGroup, vBoxTemplate, deviceType string) (*virtualbox.Machine, workstation.WorkStation, *vbox.Config, string) {
	w := workstation.NewWorkStation()
	help.ExitOnError(vbox.CheckDeps("VBoxManage"))

	conf := filepath.Join(help.UserHomeDir(), ".iotit", "virtualbox", "iotit-vbox.json")
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
			time.Sleep(20 * time.Second)
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
