package constants

const (
	ISAAX_DAEMON_DIR = "/opt/"
	ISAAX_CONF_DIR   = "/etc/"
	ISAAX_APP_DIR    = "/var/isaax/project/"

	DarwinDiskutil = "diskutil"
	UnixDD         = "dd"
	Mount          = "mount"
	DarwinUmount   = "unmountDisk"

	MountDir = "/tmp/isaax-sd/"
	Eject    = "eject"
	Umount   = "umount"

	LocaleF       string = "locale.conf"
	KeyboardF     string = "vconsole.conf"
	WPAsupplicant string = "wpa_supplicant.conf"
	InterfacesF   string = "interfaces"
	ResolveF      string = "resolv.conf"

	Language   = "LANGUAGE=%s\n"
	LocaleAll  = "LC_ALL=%s\n"
	LocaleLang = "LANG=%s\n"

	KeyMap = "KEYMAP=%s\n"

	WPAconf = `ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev

update_config=1

network={
	ssid=\"%s\"
	psk=\"%s\"
}
`

	InterfaceWLAN string = "source-directory /etc/network/interfaces.d\n" +
		"\n" +
		"auto lo\n" +
		"iface lo inet loopback\n" +
		"\n" +
		"iface eth0 inet manual\n" +
		"\n" +
		"allow-hotplug wlan0\n" +
		"iface wlan0 inet static\n" +
		"address %s\n" +
		"netmask %s\n" +
		"gateway %s\n" +
		"dns-nameservers %s\n" +
		"wpa-conf /etc/wpa_supplicant/wpa_supplicant.conf\n"

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

	Resolv string = "nameserver %s\n"
)
