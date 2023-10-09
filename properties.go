package hotplug

type DeviceClass uint

const (
	UnknownClass DeviceClass = iota

	HIDClass

	PrinterClass
)
