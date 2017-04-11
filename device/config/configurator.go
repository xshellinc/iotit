package config

import (
	"github.com/xshellinc/tools/dialogs"
	"github.com/xshellinc/tools/lib/ssh_helper"
)

// Default Configurator constants that are describe a specific configuration option
const (
	Locale = iota
	Keymap
	Wifi
	Interface
	Dns
)

// consts are a const strings representation
var consts = [...]string{
	"Locale",
	"Keymap",
	"Wifi",
	"Interface",
	"Dns",
}

// GetConstLiteral gets a literal from configurator.const
func GetConstLiteral(v int) string {
	return consts[v]
}

type (
	// Configurator is a config helper, which uses ConfigCallbackFn to store configurations for devices
	Configurator interface {
		Setup() error
		Write() error
		SetConfigFn(int, *ConfigCallbackFn)
		GetConfigFn(int) *ConfigCallbackFn
	}

	// configurator is a container of a mutual storage and order of ConfigCallbackFn
	configurator struct {
		storage map[string]interface{}
		order   []*ConfigCallbackFn
	}

	// ConfigFunction is a function with an input parameter of configurator's `storage`
	ConfigFunction func(map[string]interface{}) error

	// ConfigCallbackFn is an entity with Config and Apply function
	ConfigCallbackFn struct {
		Config ConfigFunction
		Apply  ConfigFunction
	}
)

// NewDefault creates a default Configurator
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

// Creates a new ConfigCallbackFn
func NewConfigCallbackFn(config ConfigFunction, apply ConfigFunction) *ConfigCallbackFn {
	return &ConfigCallbackFn{config, apply}
}

// Write triggers all ConfigCallbackFn Config functions
func (c *configurator) Setup() error {
	if dialogs.YesNoDialog("Would you like to configure your board?") {
		for _, o := range c.order {
			if (*o).Config == nil {
				continue
			}
			if err := o.Config(c.storage); err != nil {
				return err
			}
		}
	}

	return nil
}

// Write triggers all ConfigCallbackFn Apply functions
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

// SetConfigFn sets ConfigCallbackFn of a specified number of the array
func (c *configurator) SetConfigFn(num int, ccf *ConfigCallbackFn) {
	c.order[num] = ccf
}

// GetConfigFn returns GetConfigFn from the array by a number
func (c *configurator) GetConfigFn(num int) *ConfigCallbackFn {
	return c.order[num]
}
