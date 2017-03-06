package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device"
	"github.com/xshellinc/iotit/lib/vbox"
	"github.com/xshellinc/tools/dialogs"
)

const progName = "iotit"

const helpInfo = `
NAME:
   iotit - Flashing Tool for iot devices used by Isaax Cloud

USAGE:
   iotit [global options]

GLOBAL OPTIONS:
   -update <sd|edison> update vbox and dependencies
   -dev [device-type]  executes iotit with specified deviceType
   --help, -h           show help
   --version, -v        print the version
`

// Version string came from linker
var Version string

// Env string came from linker
var Env string

func init() {
	logrus.SetLevel(logrus.WarnLevel)
	if Env == "dev" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	f, err := os.OpenFile(fmt.Sprintf("/tmp/%s.log", progName), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		logrus.Error("error opening file: %v", err)
		return
	}

	logrus.SetOutput(f)
}

func main() {
	deviceType := flag.String("dev", "", "")
	h1 := flag.Bool("help", false, "")
	h2 := flag.Bool("h", false, "")

	v1 := flag.Bool("version", false, helpInfo)
	v2 := flag.Bool("v", false, "")

	u := flag.String("update", "", "")

	flag.Parse()

	if *h1 || *h2 {
		fmt.Println(helpInfo)
		return
	}

	if *v1 || *v2 {
		if Version == "" {
			Version = "no version"
		}
		fmt.Println(progName, Version)
		return
	}

	if *u != "" {

		if name, bool := vbox.CheckUpdate(*u); bool {
			if dialogs.YesNoDialog("Would you like to update?") {
				vbox.Update(name)
			}
		}
	}

	device.Init(*deviceType)
}
