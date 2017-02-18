package dialogs

import (
	"fmt"

	"os"

	"github.com/howeyc/gopass"
)

func WiFiPassword() string {
	retries := 3

	for retries > 0 {
		retries--

		fmt.Print("[+] WIFI password: ")
		pass, err := gopass.GetPasswdMasked()

		if err != nil {
			fmt.Println("[-] ", err.Error())
		}

		return string(pass)
	}

	fmt.Println("\n[-] You reached maximum number of retries")
	os.Exit(3)
	return ""
}
