package ping

import (
	"regexp"
	"strconv"

	"github.com/xshellinc/tools/lib/help"
)

var match, _ = regexp.Compile(`[0-9]+ packets transmitted, ([0-9])+`)

func pingIp(ip string) bool {
	out, _ := help.ExecCmd("ping", []string{"-c", "2", ip})

	sub := match.FindStringSubmatch(out)
	if len(sub) == 2 {
		n, err := strconv.Atoi(sub[1])
		if err == nil && n == 0 {
			return true
		}
	}

	return false
}
