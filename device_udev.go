//go:build linux

package hotplug

import (
	"errors"
	"runtime"
)

/*
	#cgo pkg-config: libudev
	#include <libudev.h>
	#include <stdlib.h>
*/
import "C"

type platformDevice struct {
	// prevents the udev context from being freed before the device
	listener *Listener
	udev     *C.struct_udev_device
}

func newDevice(listener *Listener, udev *C.struct_udev_device) *Device {
	syspath := C.udev_device_get_syspath(udev)

	var class DeviceClass
	for maybeClass, cond := range deviceClassCondition {
		if cond.matches(udev) {
			class = maybeClass
			break
		}
	}

	dev := &Device{}
	dev.Path = C.GoString(syspath)
	dev.Class = class
	dev.listener = listener
	dev.udev = udev

	C.udev_device_ref(udev)
	runtime.SetFinalizer(dev, freeDevice)
	return dev
}

func freeDevice(dev *Device) {
	C.udev_device_unref(dev.udev)
	dev.udev = nil
}

func (dev *Device) parent() (*Device, error) {
	parent := C.udev_device_get_parent(dev.udev)
	if parent == nil {
		return nil, errors.New("no parent")
	}

	return newDevice(dev.listener, parent), nil
}

func (dev *Device) up(class DeviceClass) (*Device, error) {
	cond := deviceClassCondition[class]

	parent := dev.udev
	for {
		parent = C.udev_device_get_parent(parent)
		if parent == nil {
			return nil, errors.New("no matching ancestor found")
		}

		if cond.matches(parent) {
			return newDevice(dev.listener, parent), nil
		}
	}
}

func (dev *Device) getSysAttrLong(attr *C.char, base int) (int, error) {
	val := C.udev_device_get_sysattr_value(dev.udev, attr)
	if val == nil {
		return 0, errors.New("attribute not found")
	}

	return (int)(C.strtol(val, nil, (C.int)(base))), nil
}

func (dev *Device) path() (string, error) {
	path := C.udev_device_get_devpath(dev.udev)
	if path == nil {
		return "", errors.New("failed to get devpath")
	}

	return C.GoString(path), nil
}

func (dev *Device) busNumber() (int, error) {
	return dev.getSysAttrLong(C.CString("busnum"), 10)
}

func (dev *Device) address() (int, error) {
	return dev.getSysAttrLong(C.CString("devnum"), 10)
}

func (dev *Device) vendorId() (int, error) {
	return dev.getSysAttrLong(C.CString("idVendor"), 16)
}

func (dev *Device) productId() (int, error) {
	return dev.getSysAttrLong(C.CString("idProduct"), 16)
}
