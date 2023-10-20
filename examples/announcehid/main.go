package main

import (
	"fmt"
	"github.com/elemecca/go-hotplug"
)

func main() {
	listener, _ := hotplug.New(
		hotplug.DevIfHid,
		func(devIf *hotplug.DeviceInterface) {
			usb, err := devIf.Device.Up(hotplug.DevUsbDevice)
			if err != nil {
				fmt.Printf("usb parent not found: %s\n", err.Error())
				return
			}

			busNumber, err := usb.BusNumber()
			if err != nil {
				fmt.Printf("failed to get bus: %s\n", err.Error())
			}

			address, err := usb.Address()
			if err != nil {
				fmt.Printf("failed to get address: %s\n", err.Error())
			}

			vendorId, err := usb.VendorId()
			if err != nil {
				fmt.Printf("failed to get vid: %s\n", err.Error())
			}

			productId, err := usb.ProductId()
			if err != nil {
				fmt.Printf("failed to get pid: %s\n", err.Error())
			}

			fmt.Printf(
				"arrive bus=%d address=%d vid=%04x pid=%04x dev=%s\n",
				busNumber, address, vendorId, productId, devIf.Path,
			)

			err = devIf.OnDetach(func() {
				fmt.Printf(
					"depart bus=%d address=%d vid=%04x pid=%04x dev=%s\n",
					busNumber, address, vendorId, productId, devIf.Path,
				)
			})
			if err != nil {
				fmt.Printf(
					"failed to register detach listener %s\n",
					err.Error(),
				)
			}
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
