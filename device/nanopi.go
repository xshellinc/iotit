package device

import "fmt"

type nanoPi struct {
	*sdFlasher
}

func (d *nanoPi) Configure() error {
	fmt.Println("nconfig")
	return nil
}
