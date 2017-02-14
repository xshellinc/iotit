package main

import (
	"flag"

	"github.com/xshellinc/iotit/device"

	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/dialogs"
	"github.com/xshellinc/iotit/lib/vbox"
)

const ProgName = "iotit"

const helpInfo = `
NAME:
   iotit - Flashing Tool for iot devices used by Isaax Cloud

USAGE:
   iotit [global options]

GLOBAL OPTIONS:
   -update <sd|edison> update vbox and dependencies
   -update <sd|edison> update vbox and dependencies
   -dev [device-type]  executes iotit with specified deviceType
   -help, -h           show help
   -version, -v        print the version
`

var Version string

func init() {
	f, err := os.OpenFile(fmt.Sprintf("/tmp/%s.log", ProgName), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	logrus.SetLevel(logrus.DebugLevel)
	if err != nil {
		logrus.Error("error opening file: %v", err)
		return
	}
	logrus.SetOutput(f)
}

func main() {
	deviceType := flag.String("dev", "", "-dev=[device-type]")
	h1 := flag.Bool("help", false, helpInfo)
	h2 := flag.Bool("h", false, helpInfo)

	v1 := flag.Bool("version", false, helpInfo)
	v2 := flag.Bool("v", false, helpInfo)

	u := flag.String("update", "", helpInfo)

	flag.Parse()

	if *h1 || *h2 {
		fmt.Println(helpInfo)
		return
	}

	if *v1 || *v2 {
		if Version == "" {
			Version = "-1 not set"
		}
		fmt.Println(ProgName, Version)
		return
	}

	if *u != "" {

		if name, bool := vbox.CheckUpdate(*u); bool {
			if dialogs.YesNoDialog("Would you like to update?") {
				vbox.VboxUpdate(name)
			}
		}
	}

	device.DeviceInit(*deviceType)
}
