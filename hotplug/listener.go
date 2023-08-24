package hotplug

import "errors"

type Listener struct {
	enabled bool
	listenerData
}

func (l *Listener) FilterIncludeBus() error {
	if l.enabled {
		return errors.New("can't adjust filters while listener is enabled")
	}

	return nil
}

func (l *Listener) FilterIncludeDeviceType(devType DeviceType) error {
	if l.enabled {
		return errors.New("can't adjust filters while listener is enabled")
	}

	return nil
}

func (l *Listener) Enable() error {
	if l.enabled {
		return errors.New("listener is already enabled")
	}

	l.enabled = true
	return l.enable()
}

func (l *Listener) Disable() error {
	if l.enabled {
		return errors.New("listener is not enabled")
	}

	err := l.disable()
	if err != nil {
		return err
	}

	l.enabled = false
	return nil
}

// Enumerate calls the callback function for each device present in the system.
func (l *Listener) Enumerate() error {
	return l.enumerate()
}
