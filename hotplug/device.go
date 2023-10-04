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

func (dev *Device) VendorId() (string, error) {
	return dev.vendorId()
}

func (dev *Device) ProductId() (string, error) {
	return dev.productId()
}

func (dev *Device) SerialNumber() (string, error) {
	return dev.serialNumber()
}
