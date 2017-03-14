package main

import (
	"flag"
	"fmt"
	"os"

	"runtime"

	"strings"

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

	f, err := os.OpenFile(fmt.Sprintf("/tmp/%s.log", progName), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		logrus.Errorf("error opening file: %v", err)
		return
	}

	logrus.SetOutput(f)

	initCommands()
}

func main() {
	deviceType := flag.String("dev", "", "")

	flag.Parse()

	commandsHandler(flag.Args())
	device.Init(*deviceType)
}

func commandsHandler(args []string) {
	for _, arg := range args {
		if c, ok := commands[arg]; ok {
			c()
			os.Exit(0)
		}
	}
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
			fmt.Println("cp", p, installPath+progName)

			sudo.Exec(sudo.InputMaskedPassword, nil, "cp", p, installPath+progName)
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

		sudo.Exec(sudo.InputMaskedPassword, nil, "rm", installPath+progName)
	}

	upd := func() {
		if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
			fmt.Println("[+] Software is not installed, please install it globally first: `" + progName + " gl`")
			return
		}

		fmt.Println("[+] Current os: ", runtime.GOOS, runtime.GOARCH)
		h1, err := repo.CheckIoTItMD5(runtime.GOOS, runtime.GOARCH)
		if err != nil {
			fmt.Println("[-] ", err)
			return
		}

		h2, err := help.HashFileMD5(installPath + progName)
		if err != nil {
			fmt.Println("[-] ", err)
			return
		}

		if !strings.EqualFold(h1, h2) {
			// @todo download process
		}

		fmt.Println("[+] You may need to enter your user password")

		sudo.Exec(sudo.InputMaskedPassword, nil, "md5sum", installPath+progName)

		if name, b := vbox.CheckUpdate("sd"); b {
			if dialogs.YesNoDialog("Would you like to update sdVbox?") {
				vbox.Update(name)
			}
		}

		if name, b := vbox.CheckUpdate("edison"); b {
			if dialogs.YesNoDialog("Would you like to update edisonVbox?") {
				vbox.Update(name)
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
