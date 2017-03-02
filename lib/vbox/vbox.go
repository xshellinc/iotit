package vbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	virtualbox "github.com/riobard/go-virtualbox"
	"github.com/xshellinc/iotit/lib/constant"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
)

type (
	// Config represents Vbox parameters with ssh and http configurations
	Config struct {
		Name        string     `json:"name"`
		UUID        string     `json:"uuid"`
		Template    string     `json:"template"`
		Device      string     `json:"device"`
		Description string     `json:"description"`
		Option      ArchConfig `json:"option"`
		SSH         SSHConfig  `json:"ssh"`
		HTTP        HTTPConfig `json:"http"`
	}

	// ArchConfig represents basic VM settings
	ArchConfig struct {
		CPU    uint          `json:"cpu"`
		Memory uint          `json:"memory"`
		USB    USBController `json:"usb"`
	}

	// USBController represents USB settings
	USBController struct {
		USB     OnOff             `json:"vc"`
		USBType USBTypeController `json:"type"`
	}

	// USBTypeController represents USB type settings
	USBTypeController struct {
		EHCI OnOff `json:"2.0"`
		XHCI OnOff `json:"3.0"`
	}

	// SSHConfig represents SSH forwarding settings
	SSHConfig struct {
		IP       string `json:"ip"`
		User     string `json:"user"`
		Password string `json:"password"`
		Port     string `json:"port"`
	}

	// HTTPConfig represents HTTP forwarding settings
	HTTPConfig struct {
		URL  string `json:"url"`
		Port string `json:"url"`
	}

	// OnOff is just a bool with Stringer interface
	OnOff bool
)

// String returns "on" or "off"
func (o OnOff) String() string {
	if o {
		return "on"
	}
	return "off"
}

// @todo replace with Help
func exit(err error) {
	if err != nil {
		log.Error("erro msg:", err.Error())
		fmt.Println("[-] Error: ", err.Error())
		fmt.Println("[-] Exiting with exit status 1 ...")
		os.Exit(1)
	}
}

// NewConfig returns new VirtualBox wrapper, containing helper functions to copy into vbox and dowload from it
// Run commands over ssh and get Virtual box configuration files
func NewConfig(template, device string) *Config {
	err := CheckMachine(template, device)
	exit(err)
	m, err := virtualbox.GetMachine(template)
	exit(err)

	return &Config{
		Name:        "",
		UUID:        m.UUID,
		Template:    m.Name,
		Device:      device,
		Description: "",
		Option: ArchConfig{
			CPU:    m.CPUs,
			Memory: m.Memory,
			USB: USBController{
				USB: m.Flag&virtualbox.FlagUSB != 0,
				USBType: USBTypeController{
					EHCI: m.Flag&virtualbox.FlagUSBEHCI != 0,
					XHCI: m.Flag&virtualbox.FlagUSBXHCI != 0,
				},
			},
		},
		SSH: SSHConfig{
			IP:       constant.TemplateIP,
			User:     constant.TemplateUser,
			Password: constant.TemplatePassword,
			Port:     constant.TemplateSSHPort,
		},
		HTTP: HTTPConfig{
			URL:  constant.TemplateURL,
			Port: constant.TemplateHTTPPort,
		},
	}
}

// RunOverSSH runs command over SSH
func (vc *Config) RunOverSSH(command string) (string, error) {
	return vc.runOverSSHWithTimeout(command, help.SshCommandTimeout)
}

// RunOverSSHExtendedPeriod runs command over SSH
func (vc *Config) RunOverSSHExtendedPeriod(command string) (string, error) {
	return vc.runOverSSHWithTimeout(command, help.SshExtendedCommandTimeout)
}

func (vc *Config) runOverSSHWithTimeout(command string, timeout int) (string, error) {
	return help.GenericRunOverSsh(command, vc.SSH.IP, vc.SSH.User, vc.SSH.Password, vc.SSH.Port,
		true, false, timeout)
}

// RunOverSSHStream runs command over SSH with stdout redirection
func (vc *Config) RunOverSSHStream(command string) (output chan string, done chan bool, err error) {
	out, eut, done, err := help.StreamEasySsh(vc.SSH.IP, vc.SSH.User, vc.SSH.Password, vc.SSH.Port, "~/.ssh/id_rsa.pub", command, help.SshExtendedCommandTimeout)
	if err != nil {
		log.Error("[-] Error running command: ", eut, ",", err.Error())
		return out, done, err
	}

	return out, done, nil
}

// SCP performs secure copy operation
func (vc *Config) SCP(src, dst string) error {
	return help.ScpWPort(src, dst, vc.SSH.IP, vc.SSH.Port, vc.SSH.User, vc.SSH.Password)
}

// Download resulting image from VirtualBox
func (vc *Config) Download(img string, wg *sync.WaitGroup) error {
	localURL := vc.HTTP.URL + ":" + vc.HTTP.Port + "/" + img
	imgName, bar, err := help.DownloadFromUrlWithAttemptsAsync(localURL, constants.TMP_DIR, constants.NUMBER_OF_RETRIES, wg)
	if err != nil {
		return err
	}
	bar.Prefix(fmt.Sprintf("[+] Download %-15s", imgName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	return nil
}

// ToJSON returns JSON representation
func (vc *Config) ToJSON() string {
	obj, err := json.Marshal(vc)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	return string(obj)
}

// WriteToFile writes JSON to file
func (vc *Config) WriteToFile(dst string) {
	if virtualbox.Exists(dst) {
		fileHandle, err := os.OpenFile(dst, os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println("[-] Error opening file: ", dst, " cause:", err.Error())
			return
		}
		writer := bufio.NewWriter(fileHandle)
		defer fileHandle.Close()
		fmt.Fprintln(writer, vc.ToJSON())
		writer.Flush()
	} else {
		fileHandle, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println("[-] Error opening file: ", dst, " cause:", err.Error())
			return
		}
		writer := bufio.NewWriter(fileHandle)
		defer fileHandle.Close()
		fmt.Fprintln(writer, vc.ToJSON())
		writer.Flush()
	}
}

// FromJSON reads JSON from file
func (vc Config) FromJSON(dst string) []Config {
	var vbox []Config
	f, _ := os.Open(dst)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		json.Unmarshal(scanner.Bytes(), &vc)
		vbox = append(vbox, vc)
	}
	return vbox
}

// Modify applies VM settings
func (vc *Config) Modify() error {
	m, err := virtualbox.GetMachine(vc.Template)
	if err != nil {
		return err
	}
	usb, ehci, xhci := vc.GetUSBs()
	m.CPUs = vc.Option.CPU
	m.Memory = vc.Option.Memory
	if usb {
		m.Flag |= virtualbox.FlagUSB
	} else {
		m.Flag &^= virtualbox.FlagUSB
	}

	if ehci {
		m.Flag |= virtualbox.FlagUSBEHCI
	} else {
		m.Flag &^= virtualbox.FlagUSBEHCI
	}

	if xhci {
		m.Flag |= virtualbox.FlagUSBXHCI
	} else {
		m.Flag &^= virtualbox.FlagUSBXHCI
	}

	if m.State != virtualbox.Poweroff {
		err := m.Poweroff()
		if err != nil {
			return err
		}
	}

	m.Description = vc.Name

	err = m.ModifySimple()
	if err != nil {
		return err
	}
	return m.Refresh()
}

// Machine wraps virtualbox.GetMachine
func (vc *Config) Machine() (*virtualbox.Machine, error) {
	m, err := virtualbox.GetMachine(vc.Template)
	return m, err
}

// Select displays VM selection dialog
func Select(vboxs []Config) Config {

	for {
		fmt.Println("[+] Available virtual machine: ")
		for i, v := range vboxs {
			fmt.Printf("\t[\x1b[34m%d\x1b[0m] \x1b[34m%s\x1b[0m - \x1b[34m%s\x1b[0m \n", i, v.Name, v.Description)
		}

		fmt.Print("[+] Please select a virtual machine: ")
		var inp int
		_, err := fmt.Scanf("%d", &inp)

		if err != nil || inp < 0 || inp >= len(vboxs) {
			fmt.Println("[-] Invalid user input")
			continue
		}

		return vboxs[inp]
	}
}

// Enable picks allowed VMs
func (vc Config) Enable(dst, template, device string) []Config {
	var (
		vboxList   = vc.FromJSON(dst)
		enableVbox []Config
	)
	for _, v := range vboxList {
		if v.Template == template && v.Device == device {
			enableVbox = append(enableVbox, v)
		}
	}
	return enableVbox
}

// GetName returns name of the VM
func (vc *Config) GetName() string {
	return vc.Name
}

// GetDescription returns description of the VM
func (vc *Config) GetDescription() string {
	return vc.Description
}

// GetMemory returns memory size of the VM
func (vc *Config) GetMemory() int {
	return int(vc.Option.Memory)
}

// GetCPU returns VM CPUs number
func (vc *Config) GetCPU() int {
	return int(vc.Option.CPU)
}

// GetUSBs returns USB settings
func (vc *Config) GetUSBs() (usb, ehci, xhci OnOff) {
	return vc.Option.USB.USB, vc.Option.USB.USBType.EHCI, vc.Option.USB.USBType.XHCI
}
