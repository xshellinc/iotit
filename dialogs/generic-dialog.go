package dialogs

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func GetSingleAnswer(question string, validators []ValidatorFn) string {
	reader := bufio.NewReader(os.Stdin)
	retries := 3
	fmt.Print(question)

Loop:
	for retries > 0 {
		retries--

		answer, err := reader.ReadString('\n')
		if err != nil {
			log.Error(err.Error())
			fmt.Println("[-] Could not read input string from stdin:", err.Error())
			continue
		}

		answer = strings.TrimSpace(answer)

		for _, validator := range validators {
			if !validator(answer) {
				continue Loop
			}
		}

		return answer
	}

	fmt.Println("\n[-] You reached maximum number of retries")
	os.Exit(3)

	return ""
}

func YesNoDialog(question string) bool {
	answer := GetSingleAnswer(question+" (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):", []ValidatorFn{YesNoValidator})
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}

func SelectOneDialog(question string, opts []string) int {
	retries := 3

	for i, v := range opts {
		fmt.Printf("   \x1b[34m[%d]\x1b[0m %s\n", i+1, v)
	}

	for retries > 0 {
		retries--
		fmt.Print(question)

		var inp int
		_, err := fmt.Scanf("%d", &inp)

		if err != nil || inp < 1 || inp > len(opts) {
			fmt.Println("[-] Invalid user input, ", err)
			continue
		}

		return inp - 1
	}

	fmt.Println("\n[-] You reached maximum number of retries")
	os.Exit(3)
	return 0
}
