package device

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ping"
)

// Prints sd card flashed message
func printDoneMessageSd(device, username, password string) {
	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t SD CARD READY!  \t\t\t\t\t\t\t\t   *")
	fmt.Printf("*\t\t PLEASE INSERT YOUR SD CARD TO YOUR %s \t\t\t\t\t   *\n", device)
	fmt.Println("*\t\t IF YOU HAVE NOT SET UP THE USB WIFI, PLEASE CONNECT TO ETHERNET \t\t   *")
	fmt.Printf("*\t\t SSH USERNAME:\x1b[31m%s\x1b[0m PASSWORD:\x1b[31m%s\x1b[0m \t\t\t\t\t\t\t   *\n",
		username, password)
	fmt.Println(strings.Repeat("*", 100))
}

// Prints flashed message over usb
func printDoneMessageUsb() {
	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t ALL DONE!  \t\t\t\t\t\t\t\t\t   *")
	fmt.Println(strings.Repeat("*", 100))
}

// set interfaces dialog
func setInterfaces(i *Interfaces) {

	var answ string

	if !setIP(i) {
		for {
			fmt.Print("[-] Do you want to try again. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m) ")
			fmt.Scan(&answ)

			if strings.EqualFold(answ, "y") || strings.EqualFold(answ, "yes") {
				setInterfaces(i)
				return
			} else if strings.EqualFold(answ, "n") || strings.EqualFold(answ, "no") {
				return
			} else {
				fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
			}
		}
	}

	i.Network = dialogs.GetSingleAnswer("Please enter your network: ", []dialogs.ValidatorFn{dialogs.IpAddressValidator})
	i.Gateway = dialogs.GetSingleAnswer("Please enter your gateway: ", []dialogs.ValidatorFn{dialogs.IpAddressValidator})
	i.Netmask = dialogs.GetSingleAnswer("Please enter your netmask: ", []dialogs.ValidatorFn{dialogs.IpAddressValidator})
	i.DNS = dialogs.GetSingleAnswer("Please enter your dns server: ", []dialogs.ValidatorFn{dialogs.IpAddressValidator})
}

// @todo replace by dialog
func setIP(i *Interfaces) bool {
	wg := &sync.WaitGroup{}

	loop := true
	retries := 3

	for retries > 0 && loop {
		i.Address = dialogs.GetSingleAnswer("IP address of the device: ", []dialogs.ValidatorFn{dialogs.IpAddressValidator})

		progress := make(chan bool)
		wg.Add(1)
		go func(progress chan bool) {
			defer close(progress)
			defer wg.Done()

			loop = !ping.PingIp(i.Address)
			if loop {
				fmt.Printf("\n[-] Sorry, a device with %s was already registered", i.Address)
			}
			retries--
		}(progress)
		help.WaitAndSpin("validating", progress)
		wg.Wait()
	}

	if retries == 0 {
		return false
	}

	return true
}
