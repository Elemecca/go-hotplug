package hotplug

type ListenerCallback func(iface *DeviceInterface)

type Listener struct {
	class     InterfaceClass
	callback  ListenerCallback
	listening bool
	platformListener
}

func New(
	class InterfaceClass,
	callback ListenerCallback,
) (*Listener, error) {
	l := &Listener{
		class:    class,
		callback: callback,
	}
	return l, l.init()
}

// Listen calls the ArriveCallback each time a device is connected.
func (l *Listener) Listen() error {
	return l.listen()
}

func (l *Listener) Stop() error {
	return l.stop()
}

// Enumerate calls the ArriveCallback for each device present in the system.
func (l *Listener) Enumerate() error {
	return l.enumerate()
}
