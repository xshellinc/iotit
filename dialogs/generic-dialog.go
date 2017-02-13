package dialogs

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func GetSingleAnswer(question string, validators []ValidatorFn) string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(question)
		answer, err := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if err != nil {
			fmt.Println("[-] Could not read input string from stdin:", err.Error())
			continue
		} else {
			for _, validator := range validators {
				if !validator(answer) {
					continue
				}
			}

			return answer
		}
	}
}

func YesNoDialog(question string) bool {
	answer := GetSingleAnswer(question+" (\x1b[33my/yes\x1b[0m OR \x1b[33mn/no\x1b[0m):", []ValidatorFn{YesNoValidator})
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}

func SelectOneDialog(question string, opts []string) int {

	for i, v := range opts {
		fmt.Printf("[%d] %s\n", i+1, v)
	}

	for {
		fmt.Print(question)
		var inp int
		_, err := fmt.Scanf("%d", &inp)

		if err != nil || inp < 1 || inp > len(opts) {
			fmt.Println("[-] Invalid user input, ", err)
			continue
		}

		return inp - 1
	}
}
