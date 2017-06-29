package help

import (
	"golang.org/x/sys/unix"
)

func getArch() (string, error) {
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return "", err
	}

	machine := make([]byte, 0, 65)

	for _, c := range uname.Machine {
		if c == 0 {
			break
		}
		machine = append(machine, byte(c))
	}

	return string(machine), nil
}
