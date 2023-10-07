package main

import (
	"fmt"
	"github.com/elemecca/go-hotplug"
)

func main() {
	hotplugConfig := hotplug.Config{
		OnlyClasses: []hotplug.DeviceClass{hotplug.HIDClass},
		ArriveCallback: func(dev *hotplug.Device) {
			busNumber, _ := dev.BusNumber()
			address, _ := dev.Address()
			vendorId, _ := dev.VendorId()
			productId, _ := dev.ProductId()

			fmt.Printf(
				"arrive bus=%d address=%d vid=%04x pid=%04x\n",
				busNumber, address, vendorId, productId,
			)
		},
	}

	_, _ = hotplug.Listen(hotplugConfig)

	// sleep forever and handle events
	select {}
}
