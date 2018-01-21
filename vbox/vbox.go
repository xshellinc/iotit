package vbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	virtualbox "github.com/xshellinc/go-virtualbox"
	"github.com/xshellinc/iotit/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ssh_helper"
	"path/filepath"
)

// VirtualBox location and connection details variables
const (
	VBoxName = "iotit-box"

	VBoxIP       = "localhost"
	VBoxUser     = "root"
	VBoxPassword = ""
	VBoxSSHPort  = "2222"

	VBoxConfFile = "iotit-vbox.json"
)

// vbox types
const (
	VBoxTypeDefault = iota
	VBoxTypeNew
	VBoxTypeUser
)

type (
	// Config represents Vbox parameters with ssh and http configurations
	Config struct {
		Name        string     `json:"name"`
		UUID        string     `json:"uuid"`
		Template    string     `json:"template"`
		Device      string     `json:"device"`
		Description string     `json:"description"`
		Option      ArchConfig `json:"option"`
		SSH         ssh_helper.Util
	}

	// ArchConfig represents basic VM settings
	ArchConfig struct {
		CPU    uint          `json:"cpu"`
		Memory uint          `json:"memory"`
		USB    USBController `json:"usb"`
	}

	// USBController represents USB settings
	USBController struct {
		USB     OnOff             `json:"vc"`
		USBType USBTypeController `json:"type"`
	}

	// USBTypeController represents USB type settings
	USBTypeController struct {
		EHCI OnOff `json:"2.0"`
		XHCI OnOff `json:"3.0"`
	}

	// OnOff is just a bool with Stringer interface
	OnOff bool
)

// String returns "on" or "off"
func (o OnOff) String() string {
	if o {
		return "on"
	}
	return "off"
}

// NewConfig returns new VirtualBox wrapper, containing helper functions to copy into vbox and dowload from it
// Run commands over ssh and get Virtual box configuration files
func NewConfig(device string) *Config {
	err := CheckMachine(VBoxName)
	help.ExitOnError(err)
	m, err := virtualbox.GetMachine(VBoxName)

	conf := Config{
		Name:        "",
		UUID:        m.UUID,
		Template:    m.Name,
		Device:      device,
		Description: "",
		Option: ArchConfig{
			CPU:    m.CPUs,
			Memory: m.Memory,
			USB: USBController{
				USB: m.Flag&virtualbox.FlagUSB != 0,
				USBType: USBTypeController{
					EHCI: m.Flag&virtualbox.FlagUSBEHCI != 0,
					XHCI: m.Flag&virtualbox.FlagUSBXHCI != 0,
				},
			},
		},
		SSH: ssh_helper.New(VBoxIP, VBoxUser, VBoxPassword, VBoxSSHPort),
	}

	return &conf
}

// ToJSON returns JSON representation
func (vc *Config) ToJSON() string {
	obj, err := json.Marshal(vc)
	if err != nil {
		log.WithField("config", vc).Error(err)
		return ""
	}
	return string(obj)
}

// WriteToFile writes JSON to file
func (vc *Config) WriteToFile(dst string) {
	if virtualbox.Exists(dst) {
		fileHandle, err := os.OpenFile(dst, os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println("[-] Error opening file: ", dst, " cause:", err.Error())
			return
		}
		writer := bufio.NewWriter(fileHandle)
		defer fileHandle.Close()
		fmt.Fprintln(writer, vc.ToJSON())
		writer.Flush()
	} else {
		fileHandle, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println("[-] Error opening file: ", dst, " cause:", err.Error())
			return
		}
		writer := bufio.NewWriter(fileHandle)
		defer fileHandle.Close()
		json := vc.ToJSON()
		if json != "" {
			fmt.Fprintln(writer, vc.ToJSON())
		}
		writer.Flush()
	}
}

// FromJSON reads JSON from file
func (vc Config) FromJSON(dst string) []Config {
	var vbox []Config
	f, _ := os.Open(dst)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		json.Unmarshal(scanner.Bytes(), &vc)
		vbox = append(vbox, vc)
	}
	return vbox
}

// Modify applies VM settings
func (vc *Config) Modify() error {
	m, err := virtualbox.GetMachine(vc.Template)
	if err != nil {
		return err
	}
	usb, ehci, xhci := vc.GetUSBs()
	m.CPUs = vc.Option.CPU
	m.Memory = vc.Option.Memory
	if usb {
		m.Flag |= virtualbox.FlagUSB
	} else {
		m.Flag &^= virtualbox.FlagUSB
	}

	if ehci {
		m.Flag |= virtualbox.FlagUSBEHCI
	} else {
		m.Flag &^= virtualbox.FlagUSBEHCI
	}

	if xhci {
		m.Flag |= virtualbox.FlagUSBXHCI
	} else {
		m.Flag &^= virtualbox.FlagUSBXHCI
	}

	if m.State != virtualbox.Poweroff {
		err := m.Poweroff()
		if err != nil {
			return err
		}
	}

	m.Description = vc.Name

	err = m.ModifySimple()
	if err != nil {
		return err
	}
	return m.Refresh()
}

// Machine wraps virtualbox.GetMachine
func (vc *Config) Machine() (*virtualbox.Machine, error) {
	m, err := virtualbox.GetMachine(vc.Template)
	return m, err
}

// Enable picks allowed VMs
func (vc Config) Enable(dst, template, device string) []Config {
	var (
		vboxList   = vc.FromJSON(dst)
		enableVbox []Config
	)
	for _, v := range vboxList {
		if v.Template == template && v.Device == device {
			enableVbox = append(enableVbox, v)
		}
	}
	return enableVbox
}

// GetName returns name of the VM
func (vc *Config) GetName() string {
	return vc.Name
}

// GetDescription returns description of the VM
func (vc *Config) GetDescription() string {
	return vc.Description
}

// GetMemory returns memory size of the VM
func (vc *Config) GetMemory() int {
	return int(vc.Option.Memory)
}

// GetCPU returns VM CPUs number
func (vc *Config) GetCPU() int {
	return int(vc.Option.CPU)
}

// GetUSBs returns USB settings
func (vc *Config) GetUSBs() (usb, ehci, xhci OnOff) {
	return vc.Option.USB.USB, vc.Option.USB.USBType.EHCI, vc.Option.USB.USBType.XHCI
}

// Stop stops VM
func (vc *Config) Stop(quiet bool) error {
	m, err := virtualbox.GetMachine(vc.UUID)
	if err != nil {
		return err
	}

	if !quiet && !dialogs.YesNoDialog("Would you like to stop the virtual machine?") {
		return nil
	}

	fmt.Println("[+] Stopping virtual machine")
	if err := m.Poweroff(); err != nil {
		return err
	}

	return nil
}

// Virtualbox dialogs
func onoff() OnOff {
	var a = []string{"on", "off"}

	n := dialogs.SelectOneDialog("Please select an option: ", a)
	return OnOff(n == 0)
}

// NameDialog asks for VM name
func (vc *Config) NameDialog() {
	if vc.Name != "" {
		fmt.Printf("[+] Your VB name set to \x1b[34m%s\x1b[0m: \n", vc.Name)
	} else {
		vc.Name = uuid.New()
		fmt.Printf("[+] Your VB name is generated \x1b[34m%s\x1b[0m: \n", vc.Name)
	}

	if dialogs.YesNoDialog("Would you like to change virtual machine name?") {
		vc.Name = dialogs.GetSingleAnswer("Enter name: ", dialogs.EmptyStringValidator)
	}
}

// DescriptionDialog asks for VM description
func (vc *Config) DescriptionDialog() {
	if vc.Description != "" {
		fmt.Printf("[+] Your VB description set to \x1b[34m%s\x1b[0m: \n", vc.Description)
	}
	if dialogs.YesNoDialog("Would you like to change virtual machine description?") {
		vc.Description = dialogs.GetSingleAnswer("Enter description: ")
	}
}

// MemoryDialog asks for VM memory size
func (vc *Config) MemoryDialog() {
	fmt.Printf("[+] Your VB memory set to \x1b[34m%d\x1b[0m MB: \n", int(vc.Option.Memory))

	if dialogs.YesNoDialog("Would you like to change virtual machine memory?") {

		if vc.Device == constants.DEVICE_TYPE_EDISON {
			fmt.Println("[+] WARNING, memory size should be \x1b[34m1024\x1b[0m MB or more!")
		}
		vc.Option.Memory = uint(dialogs.GetSingleNumber("Memory size: ", dialogs.PositiveNumber))
	}
}

// CPUDialog asks for VM CPUs number
func (vc *Config) CPUDialog() {
	fmt.Printf("[+] Your VB number of cpu set to \x1b[34m%d\x1b[0m: \n", int(vc.Option.CPU))

	if dialogs.YesNoDialog("Would you like to change the number of virtual processors?") {
		vc.Option.CPU = uint(dialogs.GetSingleNumber("Number of processors: ", dialogs.PositiveNumber))
	}
}

// USBDialog asks for VM USB settings
func (vc *Config) USBDialog() {
	usb, ehci, xhci := vc.GetUSBs()
	fmt.Printf("[+] Your VB USB Controller set to { ohci USB 1.0:\x1b[34m%v\x1b[0m | ehci USB 2.0:\x1b[34m%v\x1b[0m | xhci USB 3.0:\x1b[34m%v\x1b[0m } \n",
		usb, ehci, xhci)

	if dialogs.YesNoDialog("Would you like to change virtual machine usb type?") {
		if vc.Device == constants.DEVICE_TYPE_EDISON {
			fmt.Println("[+] WARNING, if you set the USB type to \x1b[34m3.0\x1b[0m, it will be faster, but device init may fail.")
		}
		fmt.Println("[+] ohci USB 1.0: ")
		vc.Option.USB.USB = onoff()

		fmt.Println("[+] ehci USB 2.0: ")
		vc.Option.USB.USBType.EHCI = onoff()

		fmt.Println("[+] xhci USB 3.0: ")
		vc.Option.USB.USBType.XHCI = onoff()
	}
}

// GetVbox applies default virtualbox specs or create new
func (vc *Config) GetVbox(device string, quiet bool) (*virtualbox.Machine, error) {
	conf := filepath.Join(repo.VboxDir, VBoxConfFile)
	log.WithField("path", conf).Debug("vbox config")
	err := StopMachines(quiet)
	help.ExitOnError(err)

	a, err := virtualbox.GetMachine("iotit-box")

	// Checks if the iotit box is running and skips setting section
	if a.State == virtualbox.Running {
		return a, err
	}

	// vboxs := vc.Enable(conf, VBoxName, device)
	n := VBoxTypeDefault
VBoxInit:

	// disable profile selection
	// if !quiet {
	//     n = selectVboxPreset(conf, vboxs)
	// }

	switch n {
	case VBoxTypeNew:
		// set up configuration
		vc.NameDialog()
		vc.DescriptionDialog()
		vc.MemoryDialog()
		vc.CPUDialog()
		vc.USBDialog()
		vc.WriteToFile(conf)

		// select virtual machine
		fallthrough
	case VBoxTypeUser:
		// select virtual machine
		vboxs := vc.Enable(conf, VBoxName, device)
		index := selectVM(vboxs)
		if index < 0 {
			goto VBoxInit
		}
		result := vboxs[index]

		// modify virtual machine
		err := result.Modify()
		help.ExitOnError(err)

		// get virtual machine
		m, err := result.Machine()
		return m, err

	default:
		fallthrough
	case VBoxTypeDefault:
		m, err := virtualbox.GetMachine(VBoxName)
		return m, err
	}
}

// Select option of virtualboxes, default uses default parameters of virtualbox image, others modifies vbox spec
// the name of vbox doesn't change
func selectVboxPreset(conf string, vc []Config) int {
	opts := []string{
		"Use default vbox preset",
		"Create a new vbox preset",
		"Use saved vbox preset",
	}
	optTypes := []int{
		VBoxTypeDefault,
		VBoxTypeNew,
		VBoxTypeUser,
	}
	n := len(opts)

	if _, err := os.Stat(conf); os.IsNotExist(err) || vc == nil {
		n--
	}

	return optTypes[dialogs.SelectOneDialog("Please select an option: ", opts[:n])]
}

// selectVM displays VM selection dialog
func selectVM(vboxs []Config) int {

	opts := make([]string, len(vboxs))
	for i, v := range vboxs {
		opts[i] = fmt.Sprintf("\t"+dialogs.PrintColored("%s")+" - "+dialogs.PrintColored("%s"), v.Name, v.Description)
	}

	fmt.Println("[+] Available virtual machines: ")
	return dialogs.SelectOneDialogWithBack("Please select virtual machine: ", opts)
}
