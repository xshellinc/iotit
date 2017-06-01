package vbox

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/pborman/uuid"
	"github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
)

// vbox types
const (
	VBoxTypeDefault = iota
	VBoxTypeNew
	VBoxTypeUser
)

// Virtualbox dialogs
func onoff() OnOff {
	var a = []string{"on", "off"}

	n := dialogs.SelectOneDialog("Please select an option: ", a)
	return OnOff(n == 0)
}

// NameDialog asks for VM name
func (v *Config) NameDialog() {
	if v.Name != "" {
		fmt.Printf("[+] Your VB name set to \x1b[34m%s\x1b[0m: \n", v.Name)
	} else {
		v.Name = uuid.New()
		fmt.Printf("[+] Your VB name is generated \x1b[34m%s\x1b[0m: \n", v.Name)
	}

	if dialogs.YesNoDialog("Would you like to change virtual machine name?") {
		v.Name = dialogs.GetSingleAnswer("Enter name: ", dialogs.EmptyStringValidator)
	}
}

// DescriptionDialog asks for VM description
func (v *Config) DescriptionDialog() {
	if v.Description != "" {
		fmt.Printf("[+] Your VB description set to \x1b[34m%s\x1b[0m: \n", v.Description)
	}
	if dialogs.YesNoDialog("Would you like to change virtual machine description?") {
		v.Description = dialogs.GetSingleAnswer("Enter description: ")
	}
}

// MemoryDialog asks for VM memory size
func (v *Config) MemoryDialog() {
	fmt.Printf("[+] Your VB memory set to \x1b[34m%d\x1b[0m MB: \n", int(v.Option.Memory))

	if dialogs.YesNoDialog("Would you like to change virtual machine memory?") {

		if v.Device == constants.DEVICE_TYPE_EDISON {
			fmt.Println("[+] WARNING, memory size should be \x1b[34m1024\x1b[0m MB or more!")
		}
		v.Option.Memory = uint(dialogs.GetSingleNumber("Memory size: ", dialogs.PositiveNumber))
	}
}

// CPUDialog asks for VM CPUs number
func (v *Config) CPUDialog() {
	fmt.Printf("[+] Your VB number of cpu set to \x1b[34m%d\x1b[0m: \n", int(v.Option.CPU))

	if dialogs.YesNoDialog("Would you like to change the number of virtual processors?") {
		v.Option.CPU = uint(dialogs.GetSingleNumber("Number of processors: ", dialogs.PositiveNumber))
	}
}

// USBDialog asks for VM USB settings
func (v *Config) USBDialog() {
	usb, ehci, xhci := v.GetUSBs()
	fmt.Printf("[+] Your VB USB Controller set to { ohci USB 1.0:\x1b[34m%v\x1b[0m | ehci USB 2.0:\x1b[34m%v\x1b[0m | xhci USB 3.0:\x1b[34m%v\x1b[0m } \n",
		usb, ehci, xhci)

	if dialogs.YesNoDialog("Would you like to change virtual machine usb type?") {
		if v.Device == constants.DEVICE_TYPE_EDISON {
			fmt.Println("[+] WARNING, if you set the USB type to \x1b[34m3.0\x1b[0m, it will be faster, but device init may fail.")
		}
		fmt.Println("[+] ohci USB 1.0: ")
		v.Option.USB.USB = onoff()

		fmt.Println("[+] ehci USB 2.0: ")
		v.Option.USB.USBType.EHCI = onoff()

		fmt.Println("[+] xhci USB 3.0: ")
		v.Option.USB.USBType.XHCI = onoff()
	}
}

// SetVbox creates custom virtualbox specs
func SetVbox(v *Config, device string) (*virtualbox.Machine, string, string, error) {
	conf := filepath.Join(repo.VboxDir, VBoxConfFile)
	log.WithField("path", conf).Info("custom vbox config")
	err := StopMachines()
	help.ExitOnError(err)

	a, err := virtualbox.GetMachine("iotit-box")

	// Checks if the iotit box is running and skips setting section
	if a.State == virtualbox.Running {
		return a, a.Name, a.Description, err
	}

	vboxs := v.Enable(conf, VBoxName, device)
VBoxInit:
	n := selectVboxPreset(conf, vboxs)

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
		vboxs := v.Enable(conf, VBoxName, device)
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
		return m, result.GetName(), result.GetDescription(), err

	default:
		fallthrough
	case VBoxTypeDefault:
		m, err := virtualbox.GetMachine(VBoxName)
		return m, m.Name, "", err
	}
}

// Select option of virtualboxes, default uses default parameters of virtualbox image, others modifies vbox spec
// the name of vbox doesn't change
func selectVboxPreset(conf string, v []Config) int {
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

	if _, err := os.Stat(conf); os.IsNotExist(err) || v == nil {
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
