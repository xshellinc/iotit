package vbox

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pborman/uuid"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
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

// @todo check if needed
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
