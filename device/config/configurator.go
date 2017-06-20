package config

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ping"
	"github.com/xshellinc/tools/lib/ssh_helper"
	"github.com/xshellinc/tools/locale"
	"sort"
)

type (
	// configurator is a container of a mutual storage and order of CallbackFn
	configurator struct {
		storage map[string]interface{}
		order   map[string]*callbackFn
	}

	// cb is a function with an input parameter of configurator's `storage`
	cb func(map[string]interface{}) error

	// CallbackFn is an entity with Config and Apply function
	callbackFn struct {
		Config cb
		Apply  cb
	}

	// Interfaces represents network interfaces used to setup devices
	Interfaces struct {
		Address string
		Gateway string
		Netmask string
		DNS     string
	}
)

// New creates an empty Configurator
func New(ssh ssh_helper.Util) *configurator {
	storage := make(map[string]interface{})

	// default
	order := make(map[string]*callbackFn)

	storage["ssh"] = ssh

	return &configurator{storage, order}
}

// NewDefault creates a default Configurator
func NewDefault(ssh ssh_helper.Util) *configurator {
	config := New(ssh)
	// add default callbacks
	config.order[Locale] = NewCallbackFn(SetLocale, SaveLocale)
	config.order[Keymap] = NewCallbackFn(SetKeyboard, SaveKeyboard)
	config.order[Wifi] = NewCallbackFn(SetWifi, SaveWifi)
	config.order[Interface] = NewCallbackFn(SetInterface, SaveInterface)
	config.order[DNS] = NewCallbackFn(SetSecondaryDNS, SaveSecondaryDNS)
	return config
}

// NewCallbackFn creates a new CallbackFn with 2 Function parameters
func NewCallbackFn(config cb, apply cb) *callbackFn {
	return &callbackFn{config, apply}
}

// Setup triggers all CallbackFn Config functions
func (c *configurator) Setup() error {
	var keys []string
	for k := range c.order {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if (*c.order[k]).Config == nil {
			continue
		}
		if err := c.order[k].Config(c.storage); err != nil {
			return err
		}
	}

	return nil
}

// Write triggers all CallbackFn Apply functions
func (c *configurator) Write() error {
	for _, o := range c.order {
		if (*o).Apply == nil {
			continue
		}

		if err := o.Apply(c.storage); err != nil {
			return err
		}
	}

	return nil
}

// AddConfigFn
func (c *configurator) AddConfigFn(name string, ccf *callbackFn) {
	c.order[name] = ccf
}

// SetConfigFn sets CallbackFn of the specified name
func (c *configurator) SetConfigFn(name string, ccf *callbackFn) {
	c.order[name] = ccf
}

// GetConfigFn returns configuration function by it's name
func (c *configurator) GetConfigFn(name string) *callbackFn {
	return c.order[name]
}

// RemoveConfigFn removes CallbackFn from order
func (c *configurator) RemoveConfigFn(name string) {
	delete(c.order, name)
}

func (c *configurator) StoreValue(name string, value interface{}) {
	c.storage[name] = value
}

// SetLocale is a default method to with dialog to configure the locale
func SetLocale(storage map[string]interface{}) error {

	fmt.Println("[+] Default language: ", DefaultLocale)
	if dialogs.YesNoDialog("Change default language?") {
		inp := dialogs.GetSingleAnswer("New locale: ", dialogs.CreateValidatorFn(locale.ValidateLocale))

		arr := locale.GetLocale(inp)

		var l string
		if len(arr) == 1 {
			l = arr[0].Locale
		} else {
			l = arr[dialogs.SelectOneDialog("Please select a locale from a list: ", arr.Strings())].Locale
		}

		storage[Locale] = l
	}

	return nil
}

// SaveLocale is a default method to save locale into the image
func SaveLocale(storage map[string]interface{}) error {

	if _, ok := storage[Locale]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", MountDir, ISAAX_CONF_DIR, "locale.conf")
	data := fmt.Sprintf("LANGUAGE=%s\nLANG=%s\n", storage[Locale], storage[Locale])

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	fp = help.AddPathSuffix("unix", MountDir, ISAAX_CONF_DIR, "environment")
	data = fmt.Sprintf("LC_ALL=%s\n", storage[Locale])

	_, eut, err = ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetKeyboard is a default method to with dialog to configure the keymap
func SetKeyboard(storage map[string]interface{}) error {
	var (
		loc string
		ok  bool
	)

	if loc, ok = storage[Locale].(string); ok {
		if i := strings.IndexAny(loc, "_."); i >= 0 {
			loc = loc[:i]
		}
	}

	fmt.Println("[+] Default keyboard layout: us")

	if dialogs.YesNoDialog("Change default keyboard layout?") {
		inp := dialogs.GetSingleAnswer("New keyboard layout: ",
			dialogs.CreateValidatorFn(func(layout string) error { return locale.ValidateLayout(loc, layout) }))

		arr := locale.GetLayout(loc, inp)

		var l string
		if len(arr) == 1 {
			l = arr[0].Layout
		} else {
			l = arr[dialogs.SelectOneDialog("Please select a layout from a list: ", arr.Strings())].Layout
		}

		storage[Keymap] = fmt.Sprintf("KEYMAP=%s\n", l)
	}

	return nil
}

// SaveKeyboard is a default method to save KEYMAP into the image
func SaveKeyboard(storage map[string]interface{}) error {

	if _, ok := storage[Keymap]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", MountDir, ISAAX_CONF_DIR, "vconsole.conf")
	data := storage[Keymap]

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetWifi is a dialog asking to configure wpa supplicant
func SetWifi(storage map[string]interface{}) error {
	if dialogs.YesNoDialog("Would you like to configure your Wi-Fi?") {
		storage[Wifi+"_name"] = dialogs.GetSingleAnswer("WIFI SSID name: ", dialogs.EmptyStringValidator)
		storage[Wifi+"_pass"] = []byte(dialogs.WiFiPassword())
	}

	return nil
}

// SaveWifi is a default method to save wpa_supplicant for the wifi connection
func SaveWifi(storage map[string]interface{}) error {

	if _, ok := storage[Wifi+"_name"]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", MountDir, ISAAX_CONF_DIR, "wpa_supplicant", "wpa_supplicant.conf")
	data := fmt.Sprintf(WPAconf, storage[Wifi+"_name"], storage[Wifi+"_pass"])

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetInterface is a dialog asking to setup user Interfaces for the static ip functionality
func SetInterface(storage map[string]interface{}) error {
	log.WithField("type", "default").Debug("setInterface")
	device := []string{"eth0", "wlan0"}
	i := Interfaces{
		Address: "192.168.0.254",
		Netmask: "255.255.255.0",
		Gateway: "192.168.0.1",
		DNS:     "192.168.0.1",
	}

	if dialogs.YesNoDialog("Would you like to assign static IP address for your device?") {
		fmt.Println("[+] Available network interface: ")
		num := dialogs.SelectOneDialog("Please select a network interface: ", device)
		fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")

		fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
			i.Address, i.Gateway, i.Netmask, i.DNS)

		if dialogs.YesNoDialog("Change values?") {
			AskInterfaceParams(&i)
		}

		switch device[num] {
		case "eth0":
			storage[Interface] = fmt.Sprintf(InterfaceETH, i.Address, i.Netmask, i.Gateway, i.DNS)
			fmt.Println("[+]  Ethernet interface configuration was updated")
		case "wlan0":
			storage[Interface] = fmt.Sprintf(InterfaceWLAN, i.Address, i.Netmask, i.Gateway, i.DNS)
			fmt.Println("[+]  wifi interface configuration was updated")
		}

	}

	return nil
}

// SaveInterface is a default method to save user Interfaces into the image
func SaveInterface(storage map[string]interface{}) error {

	if _, ok := storage[Interface]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", MountDir, ISAAX_CONF_DIR, "network", "interfaces")

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, storage[Interface], fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetSecondaryDNS is a dialog asking to set 8.8.8.8 DNS
func SetSecondaryDNS(storage map[string]interface{}) error {
	if dialogs.YesNoDialog("Add Google DNS as a secondary NameServer") {
		storage[DNS] = true
		return nil
	}

	return nil
}

// SaveSecondaryDNS is a default method to set 8.8.8.8 as a secondary DNS
func SaveSecondaryDNS(storage map[string]interface{}) error {

	if _, ok := storage[DNS]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", MountDir, ISAAX_CONF_DIR, "dhcp", "dhclient.conf")
	command := "append domain-name-servers 8.8.8.8, 8.8.4.4;"

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" >> %s`, command, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetInterfaces is a set of dialog to set user `Interfaces`
func AskInterfaceParams(i *Interfaces) {
	loop := true
	retries := 5

	var ip string

	for retries > 0 && loop {
		job := help.NewBackgroundJob()
		ip = dialogs.GetSingleAnswer("IP address of the device: ", dialogs.IpAddressValidator)

		go func() {
			defer job.Close()
			loop = !ping.PingIp(ip) //returns false on ping success
			if loop {
				fmt.Printf("\n[-] Sorry, device with %s already exists on the network", ip)
			}

			retries--
		}()
		help.WaitJobAndSpin("Checking availability", job)
	}

	if retries == 0 {
		if dialogs.YesNoDialog("Do you want to try again?") {
			AskInterfaceParams(i)
			return
		}
	}
	i.Address = ip
	fmt.Println("[+] Using IP:", ip)
	i.Gateway = dialogs.GetSingleAnswer("Please enter your gateway: ", dialogs.IpAddressValidator)
	i.Netmask = dialogs.GetSingleAnswer("Please enter your netmask: ", dialogs.IpAddressValidator)
	i.DNS = dialogs.GetSingleAnswer("Please enter your dns server: ", dialogs.IpAddressValidator)
}
