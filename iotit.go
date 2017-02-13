package main

import (
	"flag"

	"github.com/xshellinc/iotit/device"

	"fmt"
)

const ProgName = "iotit"

const helpInfo = `
NAME:
   iotit - Flashing Tool for iot devices used by Isaax Cloud

USAGE:
   iotit [global options]

GLOBAL OPTIONS:
   -dev [device-type] executes iotit with specified deviceType
   -help, -h          show help
   -version, -v       print the version
`

var Version string

func main() {
	deviceType := flag.String("dev", "", "-dev=[device-type]")
	h1 := flag.Bool("help", false, helpInfo)
	h2 := flag.Bool("h", false, helpInfo)

	v1 := flag.Bool("version", false, helpInfo)
	v2 := flag.Bool("v", false, helpInfo)

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

	device.DeviceInit(*deviceType)
}
