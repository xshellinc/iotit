package config

import (
	"fmt"

	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ping"
)

type (
	// Interfaces represents network interfaces used to setup devices
	Interfaces struct {
		Address string
		Gateway string
		Netmask string
		DNS     string
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
)

// SetDevice interface used to setup device's locale, keyboard layout, wifi, static network interfaces
// and upload them into the image
type SetDevice interface {
	SetLocale() error
	SetKeyborad() error
	SetWifi() error
	SetInterfaces(i Interfaces) error
}

// SetInterfaces is a set of dialog to set user `Interfaces`
func SetInterfaces(i *Interfaces) {
	if !setIP(i) {
		if dialogs.YesNoDialog("Do you want to try again?") {
			SetInterfaces(i)
		}

		return
	}

	i.Gateway = dialogs.GetSingleAnswer("Please enter your gateway: ", dialogs.IpAddressValidator)
	i.Netmask = dialogs.GetSingleAnswer("Please enter your netmask: ", dialogs.IpAddressValidator)
	i.DNS = dialogs.GetSingleAnswer("Please enter your dns server: ", dialogs.IpAddressValidator)
}

// setIP detects entered ip address is already leased, by pinging it
func setIP(i *Interfaces) bool {
	loop := true
	retries := 5

	var ip string

	for retries > 0 && loop {
		job := help.NewBackgroundJob()
		ip = dialogs.GetSingleAnswer("IP address of the device: ", dialogs.IpAddressValidator)

		go func() {
			defer job.Close()

			loop = !ping.PingIp(ip)
			if loop {
				fmt.Printf("\n[-] Sorry, a device with %s was already registered", ip)
			}

			retries--
		}()
		help.WaitJobAndSpin("validating", job)
	}

	if retries == 0 {
		return false
	}

	i.Address = ip

	return true
}
