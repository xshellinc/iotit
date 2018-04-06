package config

// Default Configurator constants that are describe a specific configuration option
const (
	Locale    = "Locale"
	Keymap    = "Keymap"
	Wifi      = "Wifi"
	Interface = "Interface"
	DNS       = "DNS"
	SSH       = "SSH"
	Camera    = "Camera"

	MountDir = "/tmp/isaax-sd/"

	Language   = "LANGUAGE=%s\n"
	LocaleAll  = "LC_ALL=%s\n"
	LocaleLang = "LANG=%s\n"

	DefaultLocale = "en_US.UTF-8"

	IsaaxConfDir = "/etc/"
	TmpDir       = "/tmp/"

	InterfaceWLAN string = "auto wlan0\n" +
		"iface wlan0 inet static\n" +
		"address %s\n" +
		"netmask %s\n" +
		"gateway %s\n" +
		"dns-nameservers %s\n"

	InterfaceETH string = "auto eth0\n" +
		"iface eth0 inet static\n" +
		"address %s\n" +
		"netmask %s\n" +
		"gateway %s\n" +
		"dns-nameservers %s\n" +
		"\n" +
		"iface default inet dhcp\n"

	WPAconf = `ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
    country=us
	update_config=1

	network={
		ssid=\"%s\"
		psk=\"%s\"
        
	}
	`
)
