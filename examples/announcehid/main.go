package main

import (
	"fmt"
	"github.com/elemecca/go-hotplug"
)

func main() {
	listener, _ := hotplug.New(
		hotplug.DevIfHid,
		func(devIf *hotplug.DeviceInterface, present bool) {
			usb, err := devIf.Device.Up(hotplug.DevUsbDevice)
			if err != nil {
				fmt.Printf("usb parent not found: %s\n", err.Error())
				return
			}

			busNumber, _ := usb.BusNumber()
			address, _ := usb.Address()
			vendorId, _ := usb.VendorId()
			productId, _ := usb.ProductId()

			var evt string
			if present {
				evt = "arrive"
			} else {
				evt = "depart"
			}

			fmt.Printf(
				"%s bus=%d address=%d vid=%04x pid=%04x dev=%s\n",
				evt, busNumber, address, vendorId, productId, devIf.Path,
			)
		},
	)

	err := listener.Listen()
	if err != nil {
		fmt.Printf("failed to listen: %s\n", err.Error())
	}

	err = listener.Enumerate()
	if err != nil {
		fmt.Printf("failed to enumerate: %s\n", err.Error())
	}

	// sleep forever and handle events
	select {}
}
