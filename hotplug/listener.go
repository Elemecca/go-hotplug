package hotplug

type ListenerCallback func(device *Device)

type Listener struct {
	callback    ListenerCallback
	onlyClasses []DeviceClass
	onlyBusses  []Bus
	listenerData
}

func Listen(
	callback ListenerCallback,
	onlyClasses []DeviceClass,
	onlyBusses []Bus,
) (*Listener, error) {
	l := &Listener{
		callback:    callback,
		onlyClasses: onlyClasses,
		onlyBusses:  onlyBusses,
	}

	err := l.enable()
	if err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Listener) Close() error {
	return l.disable()
}

// Enumerate calls the callback function for each device present in the system.
func (l *Listener) Enumerate() error {
	return l.enumerate()
}
