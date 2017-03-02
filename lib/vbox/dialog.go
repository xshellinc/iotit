package vbox

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pborman/uuid"
	"github.com/xshellinc/tools/constants"
)

// Virtualbox dialogs
func onoff() OnOff {

	var inp int
	var a = []string{"on", "off"}

	for {
		for i, e := range a {
			fmt.Printf("\t[\x1b[34m%d\x1b[0m] - \x1b[34m%s\x1b[0m \n", i, e)
		}
		fmt.Print("[+] Please select a number: ")

		_, err := fmt.Scanf("%d", &inp)
		if err != nil || inp < 0 || inp >= len(a) {
			fmt.Println("[-] Invalid user input")
			continue
		}

		return OnOff(inp == 0)
	}
}

// NameDialog asks for VM name
func (v *Config) NameDialog() {

	var inp string

	if v.Name != "" {
		fmt.Printf("[+] Your VB name set to \x1b[34m%s\x1b[0m: \n", v.Name)
		fmt.Print("[+] Would you like to change virtual machine name?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	} else {
		v.Name = uuid.New()
		fmt.Printf("[+] Your VB name is generated \x1b[34m%s\x1b[0m: \n", v.Name)
		fmt.Print("[+] Would you like to change virtual machine name?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	}
Loop:
	for {
		fmt.Scanln(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			for {
				fmt.Print("[+] Enter name:")
				inp = ""
				fmt.Scanln(&inp)
				if strings.EqualFold("", inp) {
					fmt.Println("[-] Invalid user input")
					continue
				}

				v.Name = inp
				break Loop
			}
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
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
	var inp string

	fmt.Printf("[+] Your VB memory set to \x1b[34m%d\x1b[0m MB: \n", int(v.Option.Memory))
	fmt.Print("[+] Would you like to change virtual machine memory?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")

Loop:
	for {
		fmt.Scanln(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			for {
				if v.Device == constants.DEVICE_TYPE_EDISON {
					fmt.Println("[+] WARNING, please increase memory size to \x1b[34m1024\x1b[0m MB or more!")
				}
				fmt.Print("[+] Change memory. Enter number:")

				a := -1
				_, err := fmt.Scanf("%d", &a)
				if err != nil || a < 0 {
					fmt.Println("[-] Invalid user input")
					continue
				}

				v.Option.Memory = uint(a)
				break Loop
			}
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}

// CPUDialog asks for VM CPUs number
func (v *Config) CPUDialog() {

	var inp string

	fmt.Printf("[+] Your VB number of cpu set to \x1b[34m%d\x1b[0m: \n", int(v.Option.CPU))
	fmt.Print("[+] Would you like to change the number of virtual machine processor?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")

Loop:
	for {
		fmt.Scanln(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			for {
				fmt.Print("[+] Change number of processor. Enter number:")

				a := -1
				_, err := fmt.Scanf("%d", &a)
				if err != nil || a < 0 {
					fmt.Println("[-] Invalid user input")
					continue
				}

				v.Option.CPU = uint(a)
				break Loop
			}
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}

// USBDialog asks for VM USB settings
func (v *Config) USBDialog() {
	var inp string

	usb, ehci, xhci := v.GetUSBs()
	fmt.Printf("[+] Your VB USB Controller set to { USB:\x1b[34m%v\x1b[0m | USB 2.0:\x1b[34m%v\x1b[0m | USB 3.0:\x1b[34m%v\x1b[0m } \n",
		usb, ehci, xhci)

	fmt.Print("[+] Would you like to change virtual machine usb type?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")

	for {
		fmt.Scanln(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			if v.Device == constants.DEVICE_TYPE_EDISON {
				fmt.Println("[+] WARNING, if you set the USB type to \x1b[34m3.0\x1b[0m, it will be faster, but device init may fail.")
			}
			fmt.Println("[+] USB: ")
			v.Option.USB.USB = onoff()

			fmt.Println("[+] USB 2.0: ")
			v.Option.USB.USBType.EHCI = onoff()

			fmt.Println("[+] USB 3.0: ")
			v.Option.USB.USBType.XHCI = onoff()

			break
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}
