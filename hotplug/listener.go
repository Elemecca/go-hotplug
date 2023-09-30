package hotplug

import "errors"

type HotplugEvent struct {
	Device Device
	Attach bool
}

type Listener struct {
	enabled  bool
	devTypes []DeviceClass
	listenerData
}

func (l *Listener) FilterIncludeBus() error {
	if l.enabled {
		return errors.New("can't adjust filters while listener is enabled")
	}

	return nil
}

func (l *Listener) FilterIncludeDeviceType(devType DeviceClass) error {
	if l.enabled {
		return errors.New("can't adjust filters while listener is enabled")
	}

	l.devTypes = append(l.devTypes, devType)
	return nil
}

func (l *Listener) Enable(callback ListenerCallback) error {
	if l.enabled {
		return errors.New("listener is already enabled")
	}

	l.callback = callback
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
