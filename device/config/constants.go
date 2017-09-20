package config

// Default Configurator constants that are describe a specific configuration option
const (
	Locale    = "Locale"
	Keymap    = "Keymap"
	Wifi      = "Wifi"
	Interface = "Interface"
	DNS       = "DNS"
	SSH       = "SSH"

	MountDir = "/tmp/isaax-sd/"

	Language   = "LANGUAGE=%s\n"
	LocaleAll  = "LC_ALL=%s\n"
	LocaleLang = "LANG=%s\n"

	DefaultLocale = "en_US.UTF-8"

	IsaaxConfDir = "/etc/"
	TmpDir       = "/tmp/"

	InterfaceWLAN string = "source-directory /etc/network/interfaces.d\n" +
		"\n" +
		"auto lo\n" +
		"iface lo inet loopback\n" +
		"\n" +
		"auto eth0\n" +
		"iface eth0 inet manual\n" +
		"\n" +
		"auto wlan0\n" +
		"iface wlan0 inet static\n" +
		"address %s\n" +
		"netmask %s\n" +
		"gateway %s\n" +
		"dns-nameservers %s\n"

	InterfaceETH string = "source-directory /etc/network/interfaces.d\n" +
		"\n" +
		"auto lo\n" +
		"iface lo inet loopback\n" +
		"\n" +
		"auto eth0\n" +
		"iface eth0 inet static\n" +
		"address %s\n" +
		"netmask %s\n" +
		"gateway %s\n" +
		"dns-nameservers %s\n" +
		"\n" +
		"iface default inet dhcp\n"

	WPAconf = `ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev

	update_config=1

	network={
		ssid=\"%s\"
		psk=\"%s\"
	}
	`
)
