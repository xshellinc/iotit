package config

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ssh_helper"
)

// SetLocale is a default method to with dialog to configure the locale
func SetLocale(storage map[string]interface{}) error {

	fmt.Println("[+] Default language: ", constants.DefaultLocale)
	if dialogs.YesNoDialog("Change default language?") {
		inp := dialogs.GetSingleAnswer("New locale: ", dialogs.CreateValidatorFn(constants.ValidateLocale))

		arr := constants.GetLocale(inp)

		var l string
		if len(arr) == 1 {
			l = arr[0].Locale
		} else {
			l = arr[dialogs.SelectOneDialog("Please select a locale from a list: ", arr.Strings())].Locale
		}

		storage[GetConstLiteral(Locale)] = l
	}

	return nil
}

// SaveLocale is a default method to save locale into the image
func SaveLocale(storage map[string]interface{}) error {

	if _, ok := storage[GetConstLiteral(Locale)]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", constants.MountDir, constants.ISAAX_CONF_DIR, constants.LocaleF)
	data := fmt.Sprintf(constants.Language+constants.LocaleLang, storage[GetConstLiteral(Locale)], storage[GetConstLiteral(Locale)])

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	fp = help.AddPathSuffix("unix", constants.MountDir, constants.ISAAX_CONF_DIR, "environment")
	data = fmt.Sprintf(constants.LocaleAll, storage[GetConstLiteral(Locale)])

	_, eut, err = ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetKeyboard is a default method to with dialog to configure the keymap
func SetKeyboard(storage map[string]interface{}) error {
	var (
		locale string
		ok     bool
	)

	if locale, ok = storage[GetConstLiteral(Locale)].(string); ok {
		if i := strings.IndexAny(locale, "_."); i >= 0 {
			locale = locale[:i]
		}
	}

	fmt.Println("[+] Default keyboard layout: ", constants.DefaultKeymap)

	if dialogs.YesNoDialog("Change default keyboard layout?") {
		inp := dialogs.GetSingleAnswer("New keyboard layout: ",
			dialogs.CreateValidatorFn(func(layout string) error { return constants.ValidateLayout(locale, layout) }))

		arr := constants.GetLayout(locale, inp)

		var l string
		if len(arr) == 1 {
			l = arr[0].Layout
		} else {
			l = arr[dialogs.SelectOneDialog("Please select a layout from a list: ", arr.Strings())].Layout
		}

		storage[GetConstLiteral(Keymap)] = fmt.Sprintf(constants.KeyMap, l)
	}

	return nil
}

// SaveKeyboard is a default method to save KEYMAP into the image
func SaveKeyboard(storage map[string]interface{}) error {

	if _, ok := storage[GetConstLiteral(Keymap)]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", constants.MountDir, constants.ISAAX_CONF_DIR, constants.KeyboardF)
	data := storage[GetConstLiteral(Keymap)]

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetWifi is a dialog asking to configure wpa supplicant
func SetWifi(storage map[string]interface{}) error {
	if dialogs.YesNoDialog("Would you like to configure your WI-Fi?") {
		storage[GetConstLiteral(Wifi)+"_name"] = dialogs.GetSingleAnswer("WIFI SSID name: ", dialogs.EmptyStringValidator)
		storage[GetConstLiteral(Wifi)+"_pass"] = []byte(dialogs.WiFiPassword())
	}

	return nil
}

// SaveWifi is a default method to save wpa_supplicant for the wifi connection
func SaveWifi(storage map[string]interface{}) error {

	if _, ok := storage[GetConstLiteral(Wifi)+"_name"]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", constants.MountDir, constants.ISAAX_CONF_DIR, "wpa_supplicant", constants.WPAsupplicant)
	data := fmt.Sprintf(constants.WPAconf, storage[GetConstLiteral(Wifi)+"_name"], storage[GetConstLiteral(Wifi)+"_pass"])

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, data, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetInterface is a dialog asking to setup user Interfaces for the static ip functionality
func SetInterface(storage map[string]interface{}) error {
	device := []string{"eth0", "wlan0"}
	i := Interfaces{
		Address: "192.168.0.254",
		Netmask: "255.255.255.0",
		Gateway: "192.168.0.1",
		DNS:     "192.168.0.1",
	}

	if dialogs.YesNoDialog("Would you like to assign static IP address for your device?") {
		fmt.Println("[+] Available network interface: ")
		num := dialogs.SelectOneDialog("Please select a network interface:", device)
		fmt.Println("[+] ********NOTE: ADJUST THESE VALUES ACCORDING TO YOUR LOCAL NETWORK CONFIGURATION********")

		for {
			fmt.Printf("[+] Current values are:\n \t[+] Address:%s\n\t[+] Gateway:%s\n\t[+] Netmask:%s\n\t[+] DNS:%s\n",
				i.Address, i.Gateway, i.Netmask, i.DNS)

			if dialogs.YesNoDialog("Change values?") {
				SetInterfaces(&i)

				switch device[num] {
				case "eth0":
					storage[GetConstLiteral(Interface)] = fmt.Sprintf(constants.InterfaceETH, i.Address, i.Netmask, i.Gateway, i.DNS)
					fmt.Println("[+]  Ethernet interface configuration was updated")
				case "wlan0":
					storage[GetConstLiteral(Interface)] = fmt.Sprintf(constants.InterfaceWLAN, i.Address, i.Netmask, i.Gateway, i.DNS)
					fmt.Println("[+]  wifi interface configuration was updated")
				}
			} else {
				break
			}
		}
	}

	return nil
}

// SaveInterface is a default method to save user Interfaces into the image
func SaveInterface(storage map[string]interface{}) error {

	if _, ok := storage[GetConstLiteral(Interface)]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", constants.MountDir, constants.ISAAX_CONF_DIR, "network", constants.InterfacesF)

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" > %s`, storage[GetConstLiteral(Interface)], fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}

// SetSecondaryDNS is a dialog asking to set 8.8.8.8 DNS
func SetSecondaryDNS(storage map[string]interface{}) error {
	if dialogs.YesNoDialog("Add Google DNS as a secondary NameServer") {
		storage[GetConstLiteral(DNS)] = true
		return nil
	}

	return nil
}

// SaveSecondaryDNS is a default method to set 8.8.8.8 as a secondary DNS
func SaveSecondaryDNS(storage map[string]interface{}) error {

	if _, ok := storage[GetConstLiteral(DNS)]; !ok {
		return nil
	}

	ssh, ok := storage["ssh"].(ssh_helper.Util)
	if !ok {
		return errors.New("Cannot get ssh config")
	}

	fp := help.AddPathSuffix("unix", constants.MountDir, constants.ISAAX_CONF_DIR, "dhcp", "dhclient.conf")
	command := "append domain-name-servers 8.8.8.8, 8.8.4.4;"

	_, eut, err := ssh.Run(fmt.Sprintf(`echo "%s" >> %s`, command, fp))
	if err != nil {
		return errors.New(err.Error() + ":" + eut)
	}

	return nil
}
