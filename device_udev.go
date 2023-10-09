//go:build linux

package hotplug

import (
	"errors"
	"runtime"
)

/*
	#cgo pkg-config: libudev
	#include <libudev.h>
*/
import "C"

type platformDevice struct {
	// prevents the udev context from being freed before the device
	listener *Listener
	udev     *C.struct_udev_device
}

func newDevice(listener *Listener, udev *C.struct_udev_device) *Device {
	dev := &Device{}
	dev.listener = listener
	dev.udev = udev
	runtime.SetFinalizer(dev, freeDevice)
	return dev
}

func freeDevice(dev *Device) {
	C.udev_device_unref(dev.udev)
	dev.udev = nil
}

func (dev *Device) path() (string, error) {
	path := C.udev_device_get_devpath(dev.udev)
	if path == nil {
		return "", errors.New("failed to get devpath")
	}

	return C.GoString(path), nil
}

func (dev *Device) busNumber() (int, error) {
	return 0, errors.New("not implemented")
}

func (dev *Device) address() (int, error) {
	return 0, errors.New("not implemented")
}

func (dev *Device) vendorId() (int, error) {
	return 0, errors.New("not implemented")
}

func (dev *Device) productId() (int, error) {
	return 0, errors.New("not implemented")
}
