package device

import "fmt"

type raspberryPi struct {
	*sdFlasher
}

func (d *raspberryPi) Configure() error {
	fmt.Println("rconfig")
	return nil
}