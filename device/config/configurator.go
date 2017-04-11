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
	DNS
)

// consts are a const strings representation
var consts = [...]string{
	"Locale",
	"Keymap",
	"Wifi",
	"Interface",
	"DNS",
}

// GetConstLiteral gets a literal from configurator.const
func GetConstLiteral(v int) string {
	return consts[v]
}

type (
	// Configurator is a config helper, which uses CallbackFn to store configurations for devices
	Configurator interface {
		Setup() error
		Write() error
		SetConfigFn(int, *CallbackFn)
		GetConfigFn(int) *CallbackFn
	}

	// configurator is a container of a mutual storage and order of CallbackFn
	configurator struct {
		storage map[string]interface{}
		order   []*CallbackFn
	}

	// Function is a function with an input parameter of configurator's `storage`
	Function func(map[string]interface{}) error

	// CallbackFn is an entity with Config and Apply function
	CallbackFn struct {
		Config Function
		Apply  Function
	}
)

// NewDefault creates a default Configurator
func NewDefault(ssh ssh_helper.Util) Configurator {

	s := make(map[string]interface{})

	// default
	c := make([]*CallbackFn, 0)
	c = append(c, NewCallbackFn(SetLocale, SaveLocale))
	c = append(c, NewCallbackFn(SetKeyboard, SaveKeyboard))
	c = append(c, NewCallbackFn(SetWifi, SaveWifi))
	c = append(c, NewCallbackFn(SetInterface, SaveInterface))
	c = append(c, NewCallbackFn(SetSecondaryDns, SaveSecondaryDns))

	s["ssh"] = ssh

	return &configurator{s, c}
}

// NewCallbackFn creates a new CallbackFn with 2 Function parameters
func NewCallbackFn(config Function, apply Function) *CallbackFn {
	return &CallbackFn{config, apply}
}

// Setup triggers all CallbackFn Config functions
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

// Write triggers all CallbackFn Apply functions
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

// SetConfigFn sets CallbackFn of a specified number of the array
func (c *configurator) SetConfigFn(num int, ccf *CallbackFn) {
	c.order[num] = ccf
}

// GetConfigFn returns GetConfigFn from the array by a number
func (c *configurator) GetConfigFn(num int) *CallbackFn {
	return c.order[num]
}
