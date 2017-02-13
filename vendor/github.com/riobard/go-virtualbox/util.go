package virtualbox

import (
	"net"
	"os"
)

// ParseIPv4Mask parses IPv4 netmask written in IP form (e.g. 255.255.255.0).
// This function should really belong to the net package.
func ParseIPv4Mask(s string) net.IPMask {
	mask := net.ParseIP(s)
	if mask == nil {
		return nil
	}
	return net.IPv4Mask(mask[12], mask[13], mask[14], mask[15])
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func SystemProperties() (string, error) {

	out, err := vbmOut("list", "systemproperties")
	if err != nil {
		return "", err
	}
	return out, nil
}
