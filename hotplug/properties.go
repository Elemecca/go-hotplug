package hotplug

type DeviceClass uint

const (
	UnknownClass DeviceClass = iota

	HIDClass

	PrinterClass
)

type Bus uint

const (
	UnknownBus Bus = iota

	USB
)
