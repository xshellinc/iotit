package virtualbox

import (
	"fmt"
	"strings"
)

// SetExtra sets extra data. Name could be "global"|<uuid>|<vmname>
func SetExtra(name, key, val string) error {
	return vbm("setextradata", name, key, val)
}

// DelExtraData deletes extra data. Name could be "global"|<uuid>|<vmname>
func DelExtra(name, key string) error {
	return vbm("setextradata", name, key)
}

// GetExtraData gets extra data. Name could be "global"|<uuid>|<vmname>
func GetExtraData(name, key string) (string, error) {
	out, err := vbmOut("getextradata", name, key)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(out, "Value: ") {
		return out[len("Value: "):], nil
	}

	return "", fmt.Errorf("Cannot get extra data for machine %s for key %s", name, key)
}
