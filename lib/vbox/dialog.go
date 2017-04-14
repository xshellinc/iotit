package vbox

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	VBoxTypeDelete
)

// Virtualbox dialogs
func onoff() OnOff {
	var a = []string{"on", "off"}

	n := dialogs.SelectOneDialog("Please select a number:", a)
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
	var inp string

	if v.Description != "" {
		fmt.Printf("[+] Your VB description set to \x1b[34m%s\x1b[0m: \n", v.Description)
		fmt.Print("[+] Would you like to change virtual machine description?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	} else {
		fmt.Print("[+] Would you like to set a description for the virtual machine?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	}

	for {
		fmt.Scanln(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			fmt.Print("[+] Enter description:")
			reader := bufio.NewReader(os.Stdin)
			description, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("[-] Invalid user input")
				continue
			}
			description = strings.TrimSpace(description)
			v.Description = description

			break
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}

// MemoryDialog asks for VM memory size
func (v *Config) MemoryDialog() {
	fmt.Printf("[+] Your VB memory set to \x1b[34m%d\x1b[0m MB: \n", int(v.Option.Memory))

	if dialogs.YesNoDialog("Would you like to change virtual machine memory?") {

		if v.Device == constants.DEVICE_TYPE_EDISON {
			fmt.Println("[+] WARNING, memory size should be \x1b[34m1024\x1b[0m MB or more!")
		}
		fmt.Print("[+] Change memory.")

		v.Option.Memory = uint(dialogs.GetSingleNumber("Enter value:", dialogs.PositiveNumber))
	}
}

// CPUDialog asks for VM CPUs number
func (v *Config) CPUDialog() {
	fmt.Printf("[+] Your VB number of cpu set to \x1b[34m%d\x1b[0m: \n", int(v.Option.CPU))

	if dialogs.YesNoDialog("Would you like to change the number of virtual machine processor?") {
		fmt.Println("[+] Change number of processor.")
		v.Option.CPU = uint(dialogs.GetSingleNumber("Enter value:", dialogs.PositiveNumber))
	}
}

// USBDialog asks for VM USB settings
func (v *Config) USBDialog() {
	usb, ehci, xhci := v.GetUSBs()
	fmt.Printf("[+] Your VB USB Controller set to { USB:\x1b[34m%v\x1b[0m | USB 2.0:\x1b[34m%v\x1b[0m | USB 3.0:\x1b[34m%v\x1b[0m } \n",
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
	conf := filepath.Join(repo.VboxDir, VBoxConf)
	err := StopMachines()
	help.ExitOnError(err)

	vboxs := v.Enable(conf, VBoxName, device)
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
		vboxs := v.Enable(conf, VBoxName, device)
		result := Select(vboxs)

		// modify virtual machine
		err := result.Modify()
		help.ExitOnError(err)

		// get virtual machine
		m, err := result.Machine()
		return m, result.GetName(), result.GetDescription(), err
	case VBoxTypeDelete:
		// select virtual machine
		vboxs := v.Enable(conf, VBoxName, device)
		result := Select(vboxs)

		// modify virtual machine
		err := result.Modify()
		help.ExitOnError(err)

		// get virtual machine
		m, err := result.Machine()
		SetVbox(v, device)
		fallthrough
	default:
		fallthrough
	case VBoxTypeDefault:
		m, err := virtualbox.GetMachine(VBoxName)
		return m, m.Name, "", err
	}
}

// Select option of virtualboxes, default uses default parameters of virtualbox image, others modifies vbox spec
// the name of vbox doesn't change
func selectVboxInit(conf string, v []Config) int {
	opts := []string{
		"Use default vbox preset",
		"Create a new vbox preset",
		"Use saved vbox preset",
		"Remove a saved vbox preset",
	}
	optTypes := []int{
		VBoxTypeDefault,
		VBoxTypeNew,
		VBoxTypeUser,
		VBoxTypeDelete,
	}
	n := len(opts)

	if _, err := os.Stat(conf); os.IsNotExist(err) || v == nil {
		n-=2
	}

	return optTypes[dialogs.SelectOneDialog("Please select an option: ", opts[:n])]
}
