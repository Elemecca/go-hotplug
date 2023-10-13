//go:build windows

package hotplug

/*
	#define WINVER 0x0602 // Windows 8
	#define UNICODE
	#include <windows.h>
*/
import "C"

var deviceClassToGuid map[InterfaceClass]C.GUID = map[InterfaceClass]C.GUID{
	// GUID_DEVINTERFACE_HID {4D1E55B2-F16F-11CF-88CB-001111000030}
	DevIfHid: C.GUID{
		0x4D1E55B2, 0xF16F, 0x11CF,
		[8]C.uchar{0x88, 0xCB, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30},
	},

	// GUID_DEVINTERFACE_PRINTER {28D78FAD-5A12-11D1-AE5B-0000F803A8C2}
	DevIfPrinter: C.GUID{
		0x28D78FAD, 0x5A12, 0x11D1,
		[8]C.uchar{0xAE, 0x5B, 0x00, 0x00, 0xF8, 0x03, 0xA8, 0xC2},
	},
}

var guidToDeviceClass map[C.GUID]InterfaceClass

func init() {
	guidToDeviceClass = make(map[C.GUID]InterfaceClass, len(deviceClassToGuid))
	for deviceClass, guid := range deviceClassToGuid {
		guidToDeviceClass[guid] = deviceClass
	}
}
