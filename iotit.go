package main

import (
	"flag"
	"fmt"
	"os"

	"log"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
)

const progName = "iotit"
const installPath = "/usr/local/bin/"

const helpInfo = `
NAME:
   iotit - Flashing Tool for iot devices used by Isaax Cloud

USAGE:
   iotit [global options] [commands]

   options and commands are not mandatory

COMMANDS:
   gl, global         install to global app environment
   un, uninstall      uninstall this app
   update             update binary and vbox images
   v, version         display current version
   h, help            display help

GLOBAL OPTIONS:
   -dev [device-type]  executes iotit with specified deviceType
`

// Version string came from linker
var Version string

// Env string came from linker
var Env string

var commands = make(map[string]func())

func init() {
	logrus.SetLevel(logrus.WarnLevel)
	if Env == "dev" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logfile := fmt.Sprintf(help.GetTempDir()+string(os.PathSeparator)+"%s.log", progName)

	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		logrus.Errorf("error opening file: %v", err)
		return
	}

	fmt.Println("Log location:", logfile)
	logrus.SetOutput(f)
	log.SetOutput(f)

	initCommands()
}

func main() {
	deviceType := flag.String("dev", "", "")

	flag.Parse()

	if commandsHandler(flag.Args()) {
		return
	}

	device.Init(*deviceType)
}

func commandsHandler(args []string) bool {
	for _, arg := range args {
		if c, ok := commands[arg]; ok {
			c()
			return true
		}
	}

	return false
}

func initCommands() {
	v := func() {
		if Version == "" {
			Version = "no version"
		}

		fmt.Println(progName, Version)
	}

	h := func() {
		fmt.Println(helpInfo)
	}

	i := func() {
		logrus.Debug("Checking ", installPath, progName)
		if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
			p, err := os.Executable()
			if err != nil {
				logrus.Fatal("[-] ", err)
			}

			fmt.Println("[+] You may need to enter your user password")
			_, eut, err := sudo.Exec(sudo.InputMaskedPassword, nil, "cp", p, installPath+progName)
			fmt.Println("[+] Copying", p, installPath+progName)
			if err != nil {
				fmt.Println("[-] Error: ", string(eut))
				return
			}

			fmt.Println("[+] Done")

			return
		}

		fmt.Println("[+] Software is already installed")
	}

	u := func() {
		logrus.Debug("Checking ", installPath, progName)
		if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
			fmt.Println("[+] Software is not installed")
			return
		}

		fmt.Println("[+] You may need to enter your user password")
		_, eut, err := sudo.Exec(sudo.InputMaskedPassword, nil, "rm", installPath+progName)
		fmt.Println("[+] Removing", installPath+progName)
		if err != nil {
			fmt.Println("[-] Error: ", string(eut))
			return
		}

		fmt.Println("[+] Done")
	}

	upd := func() {
		if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
			fmt.Println("[-] Software is not installed, please install it globally first: `" + progName + " gl`")
			return
		}

		fmt.Println("[+] Current os: ", runtime.GOOS, runtime.GOARCH)

		dir, err := repo.DownloadNewVersion(progName, Version, "/tmp")

		if err != nil {
			fmt.Println("[-] Error:", err)
			return
		}

		if dir == "" {
			fmt.Println("[+] ", progName, " is up to date")
		} else {

			fmt.Println("[+] You may need to enter your user password")

			if _, eut, err := sudo.Exec(sudo.InputMaskedPassword, nil, "mv", dir, installPath+progName); err != nil {
				fmt.Println("[-] Error:", eut)
				return
			}
			fmt.Println("[+]", progName, " is updated")
		}

		if vbox.CheckUpdate() {
			if dialogs.YesNoDialog("Would you like to update vbox?") {
				vbox.Update()
			}
		}
	}

	commands["version"] = v
	commands["v"] = v
	commands["help"] = h
	commands["h"] = h
	commands["global"] = i
	commands["gl"] = i
	commands["uninstall"] = u
	commands["un"] = u
	commands["update"] = upd
}
