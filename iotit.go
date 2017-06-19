package main

import (
	"fmt"
	"os"

	stdlog "log"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/iotit/device"
	"github.com/xshellinc/iotit/device/workstation"
	"github.com/xshellinc/iotit/lib/repo"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"

	"gopkg.in/urfave/cli.v1"
)

const progName = "iotit"
const installPath = "/usr/local/bin/"

// Version string came from linker
var Version string

// Env string came from linker
var Env = "dev"

var logfile = ""

func init() {
	log.SetLevel(log.WarnLevel)
	if Env == "dev" || runtime.GOOS == "windows" {
		log.SetLevel(log.DebugLevel)
	}
	logfile = fmt.Sprintf(help.GetTempDir()+help.Separator()+"%s.log", progName)

	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Errorf("error opening file: %v", err)
		return
	}

	log.SetOutput(f)
	stdlog.SetOutput(f)
}

func main() {
	app := cli.NewApp()
	app.Version = Version
	app.Name = progName
	app.Usage = "Flashing Tool for iot devices used by Isaax Cloud"

	app.Action = func(c *cli.Context) error {
		device.Init(c.Args().Get(0))
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:    "install",
			Aliases: []string{"gl"},
			Usage:   "Install to global app environment",
			Action: func(c *cli.Context) error {
				log.Debug("Checking ", installPath, progName)
				if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
					p, err := os.Executable()
					if err != nil {
						log.Fatal("[-] ", err)
					}

					fmt.Println("[+] You may need to enter your user password")
					_, eut, err := sudo.Exec(sudo.InputMaskedPassword, nil, "cp", p, installPath+progName)
					fmt.Println("[+] Copying", p, installPath+progName)
					if err != nil {
						fmt.Println("[-] Error: ", string(eut))
						return nil
					}

					fmt.Println("[+] Done")
					return nil
				}

				fmt.Println("[+] Software is already installed")
				return nil
			},
		},
		{
			Name:    "uninstall",
			Aliases: []string{"rm"},
			Usage:   "Uninstall iotit",
			Action: func(c *cli.Context) error {
				log.Debug("Checking ", installPath, progName)
				if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
					fmt.Println("[+] Software is not installed")
					return nil
				}

				fmt.Println("[+] You may need to enter your user password")
				_, eut, err := sudo.Exec(sudo.InputMaskedPassword, nil, "rm", installPath+progName)
				fmt.Println("[+] Removing", installPath+progName)
				if err != nil {
					fmt.Println("[-] Error: ", string(eut))
					return nil
				}

				fmt.Println("[+] Done")
				return nil
			},
		},
		{
			Name:    "update",
			Aliases: []string{"u"},
			Usage:   "Update binary and images",
			Action: func(c *cli.Context) error {
				if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
					fmt.Println("[-] Software is not installed, please install it globally first: `" + progName + " gl`")
					return nil
				}
				fmt.Println("[+] Current os: ", runtime.GOOS, runtime.GOARCH)
				dir, err := repo.DownloadNewVersion(progName, Version, help.GetTempDir())

				if err != nil {
					fmt.Println("[-] Error:", err)
					return nil
				}

				if dir == "" {
					fmt.Println("[+] ", progName, " is up to date")
				} else {
					fmt.Println("[+] You may need to enter your user password")
					if _, eut, err := sudo.Exec(sudo.InputMaskedPassword, nil, "mv", dir, installPath+progName); err != nil {
						fmt.Println("[-] Error:", eut)
						return nil
					}
					fmt.Println("[+]", progName, " is updated")
				}

				return nil
			},
		},
		{
			Name:    "log",
			Aliases: []string{"l"},
			Usage:   "Show log file location",
			Action: func(c *cli.Context) error {
				fmt.Println("Log location:", logfile)
				return nil
			},
		},
	}

	if runtime.GOOS == "windows" {
		clean := cli.Command{
			Name:  "clean",
			Usage: "*Windows only* Clean SD card partition table",
			Action: func(c *cli.Context) error {
				w := workstation.NewWorkStation()
				if err := w.CleanDisk(); err != nil {
					fmt.Println("[-] Error:", err)
					return nil
				}
				fmt.Println("[+] Disk formatted, now please reconnect the device.")
				return nil
			},
		}
		app.Commands = append(app.Commands, clean)
	}

	app.Run(os.Args)
}
