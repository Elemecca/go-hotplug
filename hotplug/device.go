package hotplug

type Device struct {
	platformDevice
}

func (dev *Device) Path() (string, error) {
	return dev.path()
}
