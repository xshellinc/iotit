package config

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ping"
	"github.com/xshellinc/tools/lib/ssh_helper"
	"github.com/xshellinc/tools/locale"
	"sort"
)

type (
	// configurator is a container of a mutual storage and order of CallbackFn
	Configurator struct {
		storage map[string]interface{}
		order   map[string]*CallbackFn
	}

	// cb is a function with an input parameter of configurator's `storage`
	cb func(map[string]interface{}) error

	// CallbackFn is an entity with Config and Apply function
	CallbackFn struct {
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
func New(ssh ssh_helper.Util) *Configurator {
	storage := make(map[string]interface{})

	// default
	order := make(map[string]*CallbackFn)

	storage["ssh"] = ssh

	return &Configurator{storage, order}
}

// NewDefault creates a default Configurator
func NewDefault(ssh ssh_helper.Util) *Configurator {
	config := New(ssh)
	config.AddConfigFn(Locale, NewCallbackFn(SetLocale, SaveLocale))
	config.AddConfigFn(Keymap, NewCallbackFn(SetKeyboard, SaveKeyboard))
	config.AddConfigFn(Wifi, NewCallbackFn(SetWifi, SaveWifi))
	config.AddConfigFn(Interface, NewCallbackFn(SetInterface, SaveInterface))
	config.AddConfigFn(DNS, NewCallbackFn(SetSecondaryDNS, SaveSecondaryDNS))
	config.AddConfigFn("Hostname", NewCallbackFn(SetHostname, SaveHostname))
	return config
}

// NewCallbackFn creates a new CallbackFn with 2 Function parameters
func NewCallbackFn(config cb, apply cb) *CallbackFn {
	return &CallbackFn{config, apply}
}

// Setup triggers all CallbackFn Config functions
func (c *Configurator) Setup() error {
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
func (c *Configurator) Write() error {
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
func (c *Configurator) AddConfigFn(name string, ccf *CallbackFn) {
	c.order[name] = ccf
}

// SetConfigFn sets CallbackFn of the specified name
func (c *Configurator) SetConfigFn(name string, ccf *CallbackFn) {
	c.order[name] = ccf
}

// GetConfigFn returns configuration function by it's name
func (c *Configurator) GetConfigFn(name string) *CallbackFn {
	return c.order[name]
}

// RemoveConfigFn removes CallbackFn from order
func (c *Configurator) RemoveConfigFn(name string) {
	delete(c.order, name)
}

// StoreValue stores value in storage
func (c *Configurator) StoreValue(name string, value interface{}) {
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
			l = arr[dialogs.SelectOneDialog("Please select a locale from the list: ", arr.Strings())].Locale
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

	fp := help.AddPathSuffix("unix", MountDir, IsaaxConfDir, "locale.conf")
	data := fmt.Sprintf("LANGUAGE=%s\nLANG=%s\n", storage[Locale], storage[Locale])

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	fp = help.AddPathSuffix("unix", MountDir, IsaaxConfDir, "environment")
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
			l = arr[dialogs.SelectOneDialog("Please select a layout from the list: ", arr.Strings())].Layout
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

	fp := help.AddPathSuffix("unix", MountDir, IsaaxConfDir, "vconsole.conf")
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

	fp := help.AddPathSuffix("unix", MountDir, IsaaxConfDir, "wpa_supplicant", "wpa_supplicant.conf")
	data := fmt.Sprintf(WPAconf, storage[Wifi+"_name"], storage[Wifi+"_pass"])

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	if _, ok := storage[Interface]; !ok {
		log.Debug("Adding wlan0 to interfaces...")
		fp := help.AddPathSuffix("unix", MountDir, IsaaxConfDir, "network", "interfaces")

		_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, `
    auto wlan0
    iface wlan0 inet dhcp
    source-directory /etc/network/interfaces.d
`, fp))
		if err != nil {
			log.WithField("eut", eut).Error(err.Error())
		}

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
		//TODO: allow user to setup several static interfaces at once
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

	fp := help.AddPathSuffix("unix", MountDir, IsaaxConfDir, "network", "interfaces")

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, storage[Interface], fp))
	if err != nil || strings.TrimSpace(eut) != "" {
		log.WithField("eut", eut).Error(err)
		return err
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

	fp := help.AddPathSuffix("unix", MountDir, IsaaxConfDir, "dhcp", "dhclient.conf")
	command := "append domain-name-servers 8.8.8.8, 8.8.4.4;"

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" >> %s`, command, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// AskInterfaceParams is a set of dialog to set user `Interfaces`
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

// SetHostname is a default method with a dialog to configure device hostname
func SetHostname(storage map[string]interface{}) error {
	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}
	fp := help.AddPathSuffix("unix", MountDir, "/etc/hostname")
	out, eut, err := ssh.Run("cat " + fp)
	if err != nil || strings.TrimSpace(eut) != "" {
		log.WithField("eut", eut).Error(err)
		return err
	}
	hostname := strings.TrimSpace(out)
	fmt.Println("[+] Default hostname: ", hostname)

	if dialogs.YesNoDialog("Do you want to change default hostname?") {
		storage["NewHostname"] = dialogs.GetSingleAnswer("New hostname: ", dialogs.EmptyStringValidator)
		storage["OldHostname"] = hostname
	}

	return nil
}

// SaveHostname is a default method to save hostname into the image
func SaveHostname(storage map[string]interface{}) error {

	if _, ok := storage["OldHostname"]; !ok {
		return nil
	}
	if _, ok := storage["NewHostname"]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	hosts := help.AddPathSuffix("unix", MountDir, "/etc/hosts")
	hostname := help.AddPathSuffix("unix", MountDir, "/etc/hostname")
	data := fmt.Sprintf("'s/%s/%s/g'", storage["OldHostname"], storage["NewHostname"])

	if _, eut, err := ssh.Run(fmt.Sprintf(`sed -i %s %s`, data, hosts)); err != nil || strings.TrimSpace(eut) != "" {
		log.WithField("eut", eut).Error(err)
		return err
	}
	if _, eut, err := ssh.Run(fmt.Sprintf(`sed -i %s %s`, data, hostname)); err != nil || strings.TrimSpace(eut) != "" {
		log.WithField("eut", eut).Error(err)
		return err
	}

	return nil
}
