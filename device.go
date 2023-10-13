package hotplug

import "C"

// A DeviceInterface describes a particular way to interact with a Device.
type DeviceInterface struct {
	Path   string
	Class  InterfaceClass
	Device *Device
}

type Device struct {
	Path  string
	Class DeviceClass
	platformDevice
}

func (dev *Device) Parent() (*Device, error) {
	return dev.parent()
}

// Up finds the nearest ancestor of this device which is of the given class.
func (dev *Device) Up(class DeviceClass) (*Device, error) {
	return dev.up(class)
}

// BusNumber is a number distinguishing the bus the device is connected to
// from other busses of the same type on the computer.
//
// The bus numbering scheme is bus-specific.
func (dev *Device) BusNumber() (int, error) {
	return dev.busNumber()
}

// Address is the address of the device on its bus.
//
// The interpretation of the address depends on the bus.
func (dev *Device) Address() (int, error) {
	return dev.address()
}

func (dev *Device) VendorId() (int, error) {
	return dev.vendorId()
}

func (dev *Device) ProductId() (int, error) {
	return dev.productId()
}
