package main

import (
	"fmt"
	"os"

	stdlog "log"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/xshellinc/iotit/device"
	"github.com/xshellinc/iotit/repo"
	"github.com/xshellinc/iotit/workstation"
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/sudo"
	"gopkg.in/urfave/cli.v1"
)

const progName = "iotit"
const installPath = "/usr/local/bin/"

// Version string came from linker
var version string

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
	app.Version = version
	app.Name = progName
	app.Usage = "Flashing Tool for IoT devices used by Isaax Cloud"

	app.Action = func(c *cli.Context) error {
		// TODO: launch gui by default
		flasher := device.New(c.Args()[:], "", false)
		if flasher == nil {
			return nil
		}
		if err := flasher.Flash(); err != nil {
			fmt.Println("[-] Error: ", err)
			return err
		}
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:    "flash",
			Aliases: []string{"f"},
			Usage:   "Flash image to the device",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "quiet, unattended, q", Usage: "Suppress questions and assume default answers"},
				cli.StringFlag{Name: "disk, d", Usage: "External disk or usb device"},
				cli.StringFlag{Name: "port, p", Usage: "Serial port for connected device. " +
					"If set to 'auto' first port will be used."},
			},
			ArgsUsage: "[device image]",
			Action: func(c *cli.Context) error {
				if c.Args().Get(0) == "help" {
					cli.ShowCommandHelp(c, "flash")
					return nil
				}
				port := c.String("port")
				disk := c.String("disk")
				if len(disk) > 0 {
					port = disk
				}
				flasher := device.New(c.Args()[:], port, c.Bool("quiet"))
				if flasher == nil {
					return nil
				}
				if err := flasher.Flash(); err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:    "configure",
			Aliases: []string{"c"},
			Usage:   "Configure image or device",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "disk, d", Usage: "External disk or usb device"},
				cli.StringFlag{Name: "port, p", Usage: "Serial port for connected device. " +
					"If set to 'auto' first port will be used."},
			},
			ArgsUsage: "[device image]",
			Action: func(c *cli.Context) error {
				port := c.String("port")
				disk := c.String("disk")
				if len(disk) > 0 {
					port = disk
				}
				flasher := device.New(c.Args()[:], port, c.Bool("quiet"))
				if flasher == nil {
					return nil
				}
				if err := flasher.Configure(); err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "List images, disks, ports",
			Subcommands: []cli.Command{
				{
					Name:  "devices",
					Usage: "List supported devices and images",
					Action: func(c *cli.Context) error {
						list := device.ListMapping()
						fmt.Println("Devices and images listed as \"name (" + dialogs.PrintColored("alias") + ")\"")
						for _, item := range list {
							fmt.Print("Type: " + item.Title)
							if len(item.Alias) > 0 {
								fmt.Print(" (" + dialogs.PrintColored(item.Alias) + ")")
							}
							fmt.Println()
							if len(item.Models) == 0 {
								fmt.Print("\tImages: ")
								for title, alias := range item.Images {
									fmt.Print(title)
									if len(alias) > 0 {
										fmt.Print(" (" + dialogs.PrintColored(alias) + ") ")
									}
								}
								fmt.Println()
							} else {
								for _, sub := range item.Models {
									fmt.Print("\tModel: " + sub.Title)
									if len(sub.Alias) > 0 {
										fmt.Print(" (" + dialogs.PrintColored(sub.Alias) + ")")
									}
									fmt.Println()
									fmt.Print("\t\tImages: ")
									for title, alias := range sub.Images {
										fmt.Print(title)
										if len(alias) > 0 {
											fmt.Print(" (" + dialogs.PrintColored(alias) + ") ")
										}
									}
									fmt.Println()
								}
							}
						}
						fmt.Println(dialogs.PrintColored("Examples"))
						fmt.Println("\tiotit flash raspi lite")
						fmt.Println("\tiotit flash nanopi2 android")
						fmt.Println("\tiotit flash esp32")
						return nil
					},
				},
				{
					Name:  "disks",
					Usage: "List external disks",
					Action: func(c *cli.Context) error {
						w := workstation.NewWorkStation("")
						w.PrintDisks()
						return nil
					},
				},
			},
		},
		{
			Name:    "install",
			Aliases: []string{"i"},
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
			Usage:   "Self-update",
			Action: func(c *cli.Context) error {
				if _, err := os.Stat(installPath + progName); os.IsNotExist(err) {
					fmt.Println("[-] Software is not installed, please install it globally first: `" + progName + " gl`")
					return nil
				}
				fmt.Println("[+] Current os: ", runtime.GOOS, runtime.GOARCH)
				dir, err := repo.DownloadNewVersion(progName, version, help.GetTempDir())

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
				w := workstation.NewWorkStation("")
				if err := w.CleanDisk(""); err != nil {
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
