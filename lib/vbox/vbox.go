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
	// Vbox parameters with ssh and http configurations
	VboxConfig struct {
		Name        string     `json:"name"`
		Uuid        string     `json:"uuid"`
		Template    string     `json:"template"`
		Device      string     `json:"device"`
		Description string     `json:"description"`
		Option      ArchConfig `json:"option"`
		Ssh         SshConfig  `json:"ssh"`
		Http        HttpConfig `json:"http"`
	}

	ArchConfig struct {
		Cpu    uint          `json:"cpu"`
		Memory uint          `json:"memory"`
		Usb    UsbController `json:"usb"`
	}

	UsbController struct {
		Usb     OnOff             `json:"self"`
		UsbType UsbTypeController `json:"type"`
	}

	UsbTypeController struct {
		Ehci OnOff `json:"2.0"`
		Xhci OnOff `json:"3.0"`
	}

	SshConfig struct {
		Ip       string `json:"ip"`
		User     string `json:"user"`
		Password string `json:"password"`
		Port     string `json:"port"`
	}
	HttpConfig struct {
		Url  string `json:"url"`
		Port string `json:"url"`
	}

	OnOff bool
)

func (o OnOff) String() string {
	if o {
		return "on"
	} else {
		return "off"
	}
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

// Virtualbox wrapper, containing helper functions to copy into vbox and dowload from it
// Run commands over ssh and get Virtual box configuration files
func NewVboxConfig(template, device string) *VboxConfig {
	err := CheckMachine(template, device)
	exit(err)
	m, err := virtualbox.GetMachine(template)
	exit(err)

	return &VboxConfig{
		Name:        "",
		Uuid:        m.UUID,
		Template:    m.Name,
		Device:      device,
		Description: "",
		Option: ArchConfig{
			Cpu:    m.CPUs,
			Memory: m.Memory,
			Usb: UsbController{
				Usb: m.Flag&virtualbox.F_usb != 0,
				UsbType: UsbTypeController{
					Ehci: m.Flag&virtualbox.F_usbehci != 0,
					Xhci: m.Flag&virtualbox.F_usbxhci != 0,
				},
			},
		},
		Ssh: SshConfig{
			Ip:       constant.TEMPLATE_IP,
			User:     constant.TEMPLATE_USER,
			Password: constant.TEMPLATE_PASSWORD,
			Port:     constant.TEMPLATE_SSH_PORT,
		},
		Http: HttpConfig{
			Url:  constant.TEMPLATE_URL,
			Port: constant.TEMPLATE_HTTP_PORT,
		},
	}
}

func (self *VboxConfig) RunOverSsh(command string) (string, error) {
	return self.runOverSshWithTimeout(command, help.SshCommandTimeout)
}

func (self *VboxConfig) RunOverSshExtendedPeriod(command string) (string, error) {
	return self.runOverSshWithTimeout(command, help.SshExtendedCommandTimeout)
}

func (self *VboxConfig) runOverSshWithTimeout(command string, timeout int) (string, error) {
	return help.GenericRunOverSsh(command, self.Ssh.Ip, self.Ssh.User, self.Ssh.Password, self.Ssh.Port,
		true, false, timeout)
}

func (self *VboxConfig) RunOverSshStream(command string) (output chan string, done chan bool, err error) {
	out, eut, done, err := help.StreamEasySsh(self.Ssh.Ip, self.Ssh.User, self.Ssh.Password, self.Ssh.Port, "~/.ssh/id_rsa.pub", command, help.SshExtendedCommandTimeout)
	if err != nil {
		log.Error("[-] Error running command: ", eut, ",", err.Error())
		return out, done, err
	}

	return out, done, nil
}

func (self *VboxConfig) Scp(src, dst string) error {
	return help.ScpWPort(src, dst, self.Ssh.Ip, self.Ssh.Port, self.Ssh.User, self.Ssh.Password)
}

func (self *VboxConfig) Download(img string, wg *sync.WaitGroup) error {
	local_url := self.Http.Url + ":" + self.Http.Port + "/" + img
	imgName, bar, err := help.DownloadFromUrlWithAttemptsAsync(local_url, constants.TMP_DIR, constants.NUMBER_OF_RETRIES, wg)
	if err != nil {
		return err
	}
	bar.Prefix(fmt.Sprintf("[+] Download %-15s", imgName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	return nil
}

func (self *VboxConfig) ToJson() string {
	obj, err := json.Marshal(self)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	return string(obj)
}

func (self *VboxConfig) WriteToFile(dst string) {
	if virtualbox.Exists(dst) {
		fileHandle, err := os.OpenFile(dst, os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println("[-] Error opening file: ", dst, " cause:", err.Error())
			return
		}
		writer := bufio.NewWriter(fileHandle)
		defer fileHandle.Close()
		fmt.Fprintln(writer, self.ToJson())
		writer.Flush()
	} else {
		fileHandle, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println("[-] Error opening file: ", dst, " cause:", err.Error())
			return
		}
		writer := bufio.NewWriter(fileHandle)
		defer fileHandle.Close()
		fmt.Fprintln(writer, self.ToJson())
		writer.Flush()
	}
}

func (self VboxConfig) FromJson(dst string) []VboxConfig {
	var vbox []VboxConfig
	f, _ := os.Open(dst)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		json.Unmarshal(scanner.Bytes(), &self)
		vbox = append(vbox, self)
	}
	return vbox
}

func (self *VboxConfig) Modify() error {
	m, err := virtualbox.GetMachine(self.Template)
	if err != nil {
		return err
	}
	usb, ehci, xhci := self.GetUsbs()
	m.CPUs = self.Option.Cpu
	m.Memory = self.Option.Memory
	if usb {
		m.Flag |= virtualbox.F_usb
	} else {
		m.Flag &^= virtualbox.F_usb
	}

	if ehci {
		m.Flag |= virtualbox.F_usbehci
	} else {
		m.Flag &^= virtualbox.F_usbehci
	}

	if xhci {
		m.Flag |= virtualbox.F_usbxhci
	} else {
		m.Flag &^= virtualbox.F_usbxhci
	}

	if m.State != virtualbox.Poweroff {
		err := m.Poweroff()
		if err != nil {
			return err
		}
	}

	m.Description = self.Name

	err = m.ModifySimple()
	if err != nil {
		return err
	}
	return m.Refresh()
}

func (self *VboxConfig) Machine() (*virtualbox.Machine, error) {
	m, err := virtualbox.GetMachine(self.Template)
	return m, err
}

func Select(vboxs []VboxConfig) VboxConfig {

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

func (self VboxConfig) Enable(dst, template, device string) []VboxConfig {
	var (
		vboxList   = self.FromJson(dst)
		enableVbox []VboxConfig
	)
	for _, v := range vboxList {
		if v.Template == template && v.Device == device {
			enableVbox = append(enableVbox, v)
		}
	}
	return enableVbox
}

func (self *VboxConfig) GetName() string {
	return self.Name
}

func (self *VboxConfig) GetDescription() string {
	return self.Description
}

func (self *VboxConfig) GetMemory() int {
	return int(self.Option.Memory)
}

func (self *VboxConfig) GetCpu() int {
	return int(self.Option.Cpu)
}

func (self *VboxConfig) GetUsbs() (usb, ehci, xhci OnOff) {
	return self.Option.Usb.Usb, self.Option.Usb.UsbType.Ehci, self.Option.Usb.UsbType.Xhci
}
