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
	if !setIP(i) {
		if dialogs.YesNoDialog("Do you want to try again?") {
			setInterfaces(i)
		}

		return
	}

	i.Network = dialogs.GetSingleAnswer("Please enter your network: ", dialogs.IpAddressValidator)
	i.Gateway = dialogs.GetSingleAnswer("Please enter your gateway: ", dialogs.IpAddressValidator)
	i.Netmask = dialogs.GetSingleAnswer("Please enter your netmask: ", dialogs.IpAddressValidator)
	i.DNS = dialogs.GetSingleAnswer("Please enter your dns server: ", dialogs.IpAddressValidator)
}

func setIP(i *Interfaces) bool {
	wg := &sync.WaitGroup{}

	loop := true
	retries := 3

	var ip string

	for retries > 0 && loop {
		ip := dialogs.GetSingleAnswer("IP address of the device: ", dialogs.IpAddressValidator)

		progress := make(chan bool)
		wg.Add(1)
		go func(progress chan bool) {
			defer close(progress)
			defer wg.Done()

			loop = !ping.PingIp(ip)
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

	i.Address = ip

	return true
}
