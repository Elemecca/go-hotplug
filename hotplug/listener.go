package hotplug

import "io"

type DeviceCallback func(device *Device)

type Config struct {
	ArriveCallback DeviceCallback
	OnlyClasses    []DeviceClass
	OnlyBusses     []Bus
}

// Listen calls the ArriveCallback each time a device is connected.
func Listen(config Config) (io.Closer, error) {
	return listen(config)
}

// Enumerate calls the ArriveCallback for each device present in the system.
func Enumerate(config Config) error {
	return enumerate(config)
}
