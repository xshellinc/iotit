package vbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	virtualbox "github.com/riobard/go-virtualbox"
	"github.com/xshellinc/tools/lib/help"
	"github.com/xshellinc/tools/lib/ssh_helper"
)

// VirtualBox location and connection details variables
const (
	VBoxName = "iotit-box"

	VBoxIP       = "localhost"
	VBoxUser     = "root"
	VBoxPassword = ""
	VBoxSSHPort  = "2222"

	VBoxConfFile = "iotit-vbox.json"
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
		SSH         ssh_helper.Util
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

// NewConfig returns new VirtualBox wrapper, containing helper functions to copy into vbox and dowload from it
// Run commands over ssh and get Virtual box configuration files
func NewConfig(device string) *Config {
	err := CheckMachine(VBoxName)
	help.ExitOnError(err)
	m, err := virtualbox.GetMachine(VBoxName)

	conf := Config{
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
		SSH: ssh_helper.New(VBoxIP, VBoxUser, VBoxPassword, VBoxSSHPort),
	}

	return &conf
}

// ToJSON returns JSON representation
func (vc *Config) ToJSON() string {
	obj, err := json.Marshal(vc)
	if err != nil {
		log.WithField("config", vc).Error(err)
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
		json := vc.ToJSON()
		if json != "" {
			fmt.Fprintln(writer, vc.ToJSON())
		}
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
