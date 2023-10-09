//go:build linux

package hotplug

import "C"

var deviceClassToSubsystem = map[DeviceClass]*C.char{
	HIDClass:     C.CString("hidraw"),
	PrinterClass: C.CString(""),
}

var deviceClassToDevtype = map[DeviceClass]*C.char{}
