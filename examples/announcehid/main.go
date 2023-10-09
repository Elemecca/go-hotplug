package main

import (
	"fmt"
	"github.com/elemecca/go-hotplug"
)

func main() {
	listener, _ := hotplug.New(
		hotplug.HIDClass,
		func(dev *hotplug.Device, present bool) {
			busNumber, _ := dev.BusNumber()
			address, _ := dev.Address()
			vendorId, _ := dev.VendorId()
			productId, _ := dev.ProductId()

			var evt string
			if present {
				evt = "arrive"
			} else {
				evt = "depart"
			}

			fmt.Printf(
				"%s bus=%d address=%d vid=%04x pid=%04x\n",
				evt, busNumber, address, vendorId, productId,
			)
		},
	)

	_ = listener.Listen()
	_ = listener.Enumerate()

	// sleep forever and handle events
	select {}
}
