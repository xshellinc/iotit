package device

import (
	"fmt"
	"github.com/xshellinc/iotit/device/workstation"
)

type SdFlasher interface {
	Flash() error
}

type sdFlasher struct {
	*deviceFlasher

	workstation *workstation.WorkStation
}

func (d *sdFlasher) Flash() error {
	fmt.Println("flashing")
	return nil
}
