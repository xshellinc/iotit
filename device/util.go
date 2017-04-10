package device

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// Prints sd card flashed message
func printDoneMessageSd(device, username, password string) {
	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("*\t\t SD CARD READY!  \t\t\t\t\t\t\t\t   *")
	fmt.Printf("*\t\t PLEASE INSERT YOUR SD CARD TO YOUR %s \t\t\t\t\t   *\n", device)
	fmt.Println("*\t\t IF YOU HAVE NOT SET UP THE USB WIFI, PLEASE CONNECT TO ETHERNET \t\t   *")
	fmt.Printf("*\t\t SSH USERNAME:\x1b[31m%s\x1b[0m PASSWORD:\x1b[31m%s\x1b[0m \t\t\t\t\t\t\t   *\n",
		username, password)
	fmt.Println(strings.Repeat("*", 100))
}

func getExtractCommand(file string) string {
	if hasAnySuffixes(file, ".tar.gz", ".tgz", ".tar.bz2", ".tbz", ".tar.xz") {
		return "tar xvf %s -C %s"
	}
	if strings.HasSuffix(file, "img.xz") {
		file = file[:len(file)-3]
		return "xz -dc %s > %s" + file + " && echo " + file
	}
	if strings.HasSuffix(file, ".zip") {
		return "unzip -o %s -d %s"
	}

	return ""
}

func hasAnySuffixes(file string, suffix ...string) bool {
	for _, s := range suffix {
		if strings.HasSuffix(file, s) {
			return true
		}
	}

	return false
}

// delete host from ssh file or any other provided
func deleteHost(fileName, host string) error {
	result := []string{}
	input, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	lines := strings.Split(string(input), "\n")
	for _, line := range lines {
		if !strings.Contains(line, host) {
			result = append(result, line)
		}
	}
	output := strings.Join(result, "\n")

	if err = ioutil.WriteFile(fileName, []byte(output), 0644); err != nil {
		return err
	}
	return nil
}
