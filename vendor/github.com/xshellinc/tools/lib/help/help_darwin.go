package help

import (
	"runtime"
)

func getArch() (string, error) {
	return runtime.GOARCH, nil
}
