package config

import (
	"fmt"

	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/ssh_helper"
)

const (
	Locale = iota
	Keymap
	Wifi
	Interface
	Dns
)

var consts = [...]string{
	"Locale",
	"Keymap",
	"Wifi",
	"Interface",
	"Dns",
}

func GetConstLiteral(v int) string {
	return consts[v]
}

type (
	Configurator interface {
		Setup() error
		Write() error
		SetConfigFn(int, *ConfigCallbackFn)
		GetConfigFn(int) *ConfigCallbackFn
	}

	configurator struct {
		storage map[string]interface{}
		order   []*ConfigCallbackFn
	}

	ConfigFunction func(map[string]interface{}) error

	ConfigCallbackFn struct {
		Config ConfigFunction
		Apply  ConfigFunction
	}
)

func NewDefault(ssh ssh_helper.Util) Configurator {

	s := make(map[string]interface{})

	// default
	c := make([]*ConfigCallbackFn, 0)
	c = append(c, NewConfigCallbackFn(ConfigLocale, SaveLocale))
	c = append(c, NewConfigCallbackFn(ConfigKeyboard, SaveKeyboard))
	c = append(c, NewConfigCallbackFn(ConfigWifi, SaveWifi))
	c = append(c, NewConfigCallbackFn(ConfigInterface, SaveInterface))
	c = append(c, NewConfigCallbackFn(ConfigSecondaryDns, SaveSecondaryDns))

	s["ssh"] = ssh

	return &configurator{s, c}
}

func NewConfigCallbackFn(config ConfigFunction, apply ConfigFunction) *ConfigCallbackFn {
	return &ConfigCallbackFn{config, apply}
}

func (c *configurator) Setup() error {
	if dialogs.YesNoDialog("Would you like to configure your board?") {
		for _, o := range c.order {
			if (*o).Config == nil {
				continue
			}
			if err := o.Config(c.storage); err != nil {
				return err
			}

			fmt.Println(c.storage)
		}
	}

	return nil
}

func (c *configurator) Write() error {
	for _, o := range c.order {
		if (*o).Apply == nil {
			continue
		}

		if err := o.Apply(c.storage); err != nil {
			return err
		}
	}

	return nil
}

func (c *configurator) SetConfigFn(num int, ccf *ConfigCallbackFn) {
	c.order[num] = ccf
}

func (c *configurator) GetConfigFn(num int) *ConfigCallbackFn {
	return c.order[num]
}
