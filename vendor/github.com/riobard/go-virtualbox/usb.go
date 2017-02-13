package virtualbox

// UsbController represents a virtualized usb controller.
type UsbController struct {
	Usb     string
	UsbType UsbTypeController
}

type UsbTypeController struct {
	Ehci string
	Xhci string
}
