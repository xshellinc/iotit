package sudo

import (
	"fmt"

	"github.com/howeyc/gopass"
)

// Masks a password input pausing a bool channel
func InputMaskedPassword(data interface{}) string {
	ch, _ := data.(chan bool)

	if ch != nil {
		ch <- false
	}

	fmt.Print("\033[K1\r[+] Enter Password: ")
	pass, _ := gopass.GetPasswdMasked()

	if ch != nil {
		ch <- true
	}

	return string(pass)
}
