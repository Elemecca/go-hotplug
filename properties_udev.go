//go:build linux

package hotplug

/*
	#cgo pkg-config: libudev
	#include <libudev.h>
	#include <string.h>
*/
import "C"

type deviceCondition struct {
	subsystem *C.char
	devtype   *C.char
	driver    *C.char

	// interfaceOnly indicates that this sysfs device is only a DeviceInterface
	// its Device is the parent sysfs device
	interfaceOnly bool
}

func (cond *deviceCondition) matches(dev *C.struct_udev_device) bool {
	subsystem := C.udev_device_get_subsystem(dev)
	if subsystem == nil || C.strcmp(cond.subsystem, subsystem) != 0 {
		return false
	}

	if cond.devtype != nil {
		devtype := C.udev_device_get_devtype(dev)
		if devtype == nil || C.strcmp(cond.devtype, devtype) != 0 {
			return false
		}
	}

	// beyond this point are properties of the device, not the interface,
	// so we need to handle interface-only nodes
	if cond.interfaceOnly {
		dev = C.udev_device_get_parent(dev)
		if dev == nil {
			return false
		}
	}

	if cond.driver != nil {
		driver := C.udev_device_get_driver(dev)
		if driver == nil || C.strcmp(cond.driver, driver) != 0 {
			return false
		}
	}

	return true
}

var interfaceClassCondition = map[InterfaceClass]*deviceCondition{
	DevIfHid: {
		subsystem:     C.CString("hidraw"),
		interfaceOnly: true,
	},
	DevIfPrinter: {
		subsystem:     C.CString("usbmisc"),
		driver:        C.CString("usblp"),
		interfaceOnly: true,
	},
}

var deviceClassCondition = map[DeviceClass]*deviceCondition{
	DevHid: {
		subsystem: C.CString("hid"),
	},
	DevUsbDevice: {
		subsystem: C.CString("usb"),
		devtype:   C.CString("usb_device"),
	},
	DevUsbInterface: {
		subsystem: C.CString("usb"),
		devtype:   C.CString("usb_interface"),
	},
}
