package dialogs

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Validator type will be called on user input
// return true if valid input, false otherwise
type ValidatorFn func(input string) bool

// return true if string is not empty and valid, false otherwise
func EmptyStringValidator(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		fmt.Println("[-] Empty input")
		return false
	}
	return true
}

// return true if parsed IP is not nil, false otherwise
func IpAddressValidator(ipAddress string) bool {
	fmt.Println("[+] Validating IP address:", ipAddress)
	ipAddress = strings.TrimSpace(ipAddress)
	re, _ := regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
	if re.MatchString(ipAddress) {
		return true
	}
	fmt.Println("[-] Not valid IP address")
	return false
}

func AppNameValidator(name string) bool {
	if name == "" {
		return true
	}
	chars := []string{"/", "\\", "?", "%", "*", ":", "|", "\"", "<", ">", ".", " ", "$"}
	for _, char := range chars {
		if strings.Contains(name, char) {
			log.Error("AppCreate func(): contains illegal char:", char)
			fmt.Println("[-] Name is not valid, contains illegal char:", char)
			return true
		}
	}
	return false
}

func YesNoValidator(answer string) bool {
	if strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes") ||
		strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no") {
		return true
	} else {
		fmt.Println("[-] Unknown user input. Please enter (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m)")
		return false
	}
}
