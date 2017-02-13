package dialogs

import (
	"fmt"
	"github.com/howeyc/gopass"
)

func WiFiPassword() string {
	fmt.Print("[+] WIFI password: ")
	pass, _ := gopass.GetPasswdMasked()
	return string(pass)
}
