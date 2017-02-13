package vbox

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pborman/uuid"
	"github.com/xshellinc/tools/constants"
)

func onoff() string {

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

		return a[inp]
	}
}

func (self *VboxConfig) NameDialog() {

	var inp string

	if self.Name != "" {
		fmt.Printf("[+] Your VB name set to \x1b[34m%s\x1b[0m: \n", self.Name)
		fmt.Print("[+] Would you like to change virtual machine name?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")
	} else {
		self.Name = uuid.New()
		fmt.Printf("[+] Your VB name is generated \x1b[34m%s\x1b[0m: \n", self.Name)
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

				self.Name = inp
				break Loop
			}
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}

func (self *VboxConfig) DescriptionDialog() {
	var inp string

	if self.Description != "" {
		fmt.Printf("[+] Your VB description set to \x1b[34m%s\x1b[0m: \n", self.Description)
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
			self.Description = description

			break
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}

func (self *VboxConfig) MemoryDialog() {
	var inp string

	fmt.Printf("[+] Your VB memory set to \x1b[34m%d\x1b[0m MB: \n", int(self.Option.Memory))
	fmt.Print("[+] Would you like to change virtual machine memory?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")

Loop:
	for {
		fmt.Scanln(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			for {
				if self.Device == constants.DEVICE_TYPE_EDISON {
					fmt.Println("[+] WARNING, please increase memory size to \x1b[34m1024\x1b[0m MB or more!")
				}
				fmt.Print("[+] Change memory. Enter number:")

				a := -1
				_, err := fmt.Scanf("%d", &a)
				if err != nil || a < 0 {
					fmt.Println("[-] Invalid user input")
					continue
				}

				self.Option.Memory = uint(a)
				break Loop
			}
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}

func (self *VboxConfig) CpuDialog() {

	var inp string

	fmt.Printf("[+] Your VB number of cpu set to \x1b[34m%d\x1b[0m: \n", int(self.Option.Cpu))
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

				self.Option.Cpu = uint(a)
				break Loop
			}
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			self.Option.Cpu = self.Option.Cpu
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}

func (self *VboxConfig) UsbDialog() {
	var inp string

	usb, ehci, xhci := self.GetUsbs()
	fmt.Printf("[+] Your VB USB Controller set to { USB:\x1b[34m%s\x1b[0m | USB 2.0:\x1b[34m%s\x1b[0m | USB 3.0:\x1b[34m%s\x1b[0m } \n",
		usb, ehci, xhci)

	fmt.Print("[+] Would you like to change virtual machine usb type?(\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):")

	for {
		fmt.Scanln(&inp)
		if strings.EqualFold(inp, "y") || strings.EqualFold(inp, "yes") {
			if self.Device == constants.DEVICE_TYPE_EDISON {
				fmt.Println("[+] WARNING, if you set the USB type to \x1b[34m3.0\x1b[0m, it will be faster, but device init may fail.")
			}
			fmt.Println("[+] USB: ")
			self.Option.Usb.Usb = onoff()
			fmt.Println("[+] USB 2.0: ")
			self.Option.Usb.UsbType.Ehci = onoff()
			fmt.Println("[+] USB 3.0: ")
			self.Option.Usb.UsbType.Xhci = onoff()
			break
		} else if strings.EqualFold(inp, "n") || strings.EqualFold(inp, "no") {
			self.Option.Usb.Usb = self.Option.Usb.Usb
			self.Option.Usb.UsbType.Ehci = self.Option.Usb.UsbType.Ehci
			self.Option.Usb.UsbType.Xhci = self.Option.Usb.UsbType.Xhci
			break
		} else {
			fmt.Print("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		}
	}
}
