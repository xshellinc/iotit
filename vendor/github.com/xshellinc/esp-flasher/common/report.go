package common

import (
	"encoding/hex"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

func Reportf(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
	log.Infof(f, args...)
}

func LimitStr(b []byte, n int) string {
	if len(b) <= n {
		return hex.EncodeToString(b)
	} else {
		return hex.EncodeToString(b[:n]) + "..."
	}
}
