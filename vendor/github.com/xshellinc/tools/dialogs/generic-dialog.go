package dialogs

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/Nerdmaster/terminal"
	log "github.com/sirupsen/logrus"
)

var p *terminal.Prompt

const Retries = 5

func readInput(prompt string) (string, error) {
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	defer terminal.Restore(0, oldState)
	p = terminal.NewPrompt(os.Stdin, os.Stdout, prompt)
	p.OnKeypress = func(e *terminal.KeyEvent) {
		if e.Key == terminal.KeyCtrlC {
			terminal.Restore(0, oldState)
			os.Exit(0)
		}
	}
	return p.ReadLine()
}

func GetSingleAnswer(question string, validators ...ValidatorFn) string {
	retries := Retries

Loop:
	for retries > 0 {
		retries--

		inp, err := readInput("[?] " + question)
		if err != nil {
			log.Error(err)
			fmt.Println("[-] Could not read input string from stdin: ", err.Error())
			fmt.Print("[?] Please try again: ")
			continue
		}

		inp = strings.TrimSpace(inp)

		for _, validator := range validators {
			if !validator(inp) {
				continue Loop
			}
		}
		return inp
	}

	fmt.Println("\n[-] You have reached maximum number of retries")
	os.Exit(3)

	return ""
}

func GetSingleNumber(question string, validators ...NumberValidatorFn) int {
	retries := Retries
	fmt.Print("[?] ", question)

Loop:
	for retries > 0 {
		retries--

		inp, err := readInput("[?] " + question)
		answer, err := strconv.Atoi(inp)
		if err != nil {
			log.Error(err)
			fmt.Println("[-] Invalid input: ", err.Error())
			continue
		}
		for _, validator := range validators {
			if !validator(answer) {
				continue Loop
			}
		}

		return answer
	}

	fmt.Println("\n[-] You have reached maximum number of retries")
	os.Exit(3)

	return 0
}

func YesNoDialog(question string) bool {
	answer := GetSingleAnswer(question+" ("+PrintColored("y/yes")+", "+PrintColored("n/no")+"): ", YesNoValidator)
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}

type YesNoAnswer int

const (
	AnswerNo YesNoAnswer = iota
	AnswerYes
	AnswerBack
)

func YesNoBackDialog(question string) YesNoAnswer {
	answer := GetSingleAnswer(question+" ("+PrintColored("y/yes")+", "+PrintColored("n/no")+" or "+PrintColored("b/back")+"): ", YesNoBackValidator)

	switch {
	case strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"):
		return AnswerYes
	case strings.EqualFold(answer, "n") || strings.EqualFold(answer, "no"):
		return AnswerNo
	default:
		return AnswerBack
	}
}

func PrintColored(str string) string {
	if runtime.GOOS == "windows" {
		return str
	} else {
		return fmt.Sprintf("\x1b[33m%s\x1b[0m", str)
	}
}

func printMenuItem(i, v interface{}) {
	fmt.Printf("   "+PrintColored("[%v]")+" %v\n", i, v)
}

func SelectOneDialog(question string, opts []string) int {
	retries := Retries

	for i, v := range opts {
		printMenuItem(i+1, v)
	}

	for retries > 0 {
		retries--
		inp, err := readInput("[?] " + question)
		if err != nil {
			fmt.Println("[-]", err.Error())
			continue
		}
		answer, err := strconv.Atoi(inp)
		if err != nil || answer < 1 || answer > len(opts) {
			fmt.Println("[-] Invalid user input, try again please.")
			continue
		}

		return answer - 1
	}

	fmt.Println("\n[-] You reached maximum number of retries")
	os.Exit(3)
	return 0
}

// SelectOneDialogWithBack returns -1 when "go back" choosen
func SelectOneDialogWithBack(question string, opts []string) int {
	retries := 3

	for i, v := range opts {
		printMenuItem(i+1, v)
	}

	printMenuItem(0, "Go Back")

	for retries > 0 {
		retries--
		inp, err := readInput("[?] " + question)
		if err != nil {
			fmt.Println("[-]", err.Error())
			continue
		}

		answer, err := strconv.Atoi(inp)
		if err != nil || answer < 0 || answer > len(opts) {
			fmt.Println("[-] Invalid input, please try again: ")
			continue
		}

		return answer - 1
	}

	fmt.Println("\n[-] You reached maximum number of retries")
	os.Exit(3)
	return 0
}

// SelectMultipleDialog returns nil when "go back" choosen
func SelectMultipleDialog(question string, opts []string, backItem bool) []int {
	retries := 3

	for i, v := range opts {
		printMenuItem(i+1, v)
	}

	printMenuItem("*", "Select All")
	if backItem {
		printMenuItem(0, "Go Back")
	}

Retry:
	for retries > 0 {
		retries--
		fmt.Println("[?]", question)
		answer, err := readInput("[?] Separate multiple numbers with comma or space: ")
		if err != nil {
			log.Error(err)
			fmt.Println("[-] Could not read input string from stdin: ", err)
			continue
		}

		fields := strings.FieldsFunc(answer, func(r rune) bool {
			return r == ',' || unicode.IsSpace(r)
		})

		out := make([]int, len(fields))
		for i, f := range fields {
			// Select all
			if f == "*" {
				out := make([]int, len(opts))
				for i := range opts {
					out[i] = i
				}
				return out
			}

			var low int
			if !backItem {
				low = 1
			}

			val, err := strconv.Atoi(f)
			if err != nil || val < low || val > len(opts) {
				var msg string
				if err != nil {
					msg = err.Error()
				}
				fmt.Println("[-] Invalid user input, ", msg, " please repeat: ")
				continue Retry
			}

			if backItem && val == 0 {
				return nil
			}

			out[i] = val - 1
		}

		return out
	}

	fmt.Println("\n[-] You reached maximum number of retries")
	os.Exit(3)
	return nil
}
