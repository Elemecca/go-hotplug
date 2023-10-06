package hotplug

type Device struct {
	platformDevice
}

func (dev *Device) Path() (string, error) {
	return dev.path()
}

func (dev *Device) Class() (DeviceClass, error) {
	return dev.class()
}

func (dev *Device) Bus() (Bus, error) {
	return dev.bus()
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
