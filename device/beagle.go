package device

import "fmt"

type beagleBone struct {
	*sdFlasher
}

func (d *beagleBone) Configure() error {
	fmt.Println("bnconfig")
	return nil
}
