package ping

import (
	"regexp"
	"strconv"

	log "github.com/sirupsen/logrus"
	"os/exec"
)

var match, _ = regexp.Compile(`Received = \d+, Lost = (\d+)`)

func pingIp(ip string) bool {
	if out, err := exec.Command("ping", "-n", "2", ip).CombinedOutput(); err != nil {
		log.Error("ping", err)
		return true
	} else {
		log.Debug(string(out))
		sub := match.FindStringSubmatch(string(out))
		if len(sub) == 2 {
			n, err := strconv.Atoi(sub[1])
			if err == nil && n == 0 {
				return false
			}
		}
	}
	return true
}
