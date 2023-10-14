//go:build windows

package hotplug

// #include "common_windows.h"
import "C"

var interfaceClassToGuid = map[InterfaceClass]C.GUID{
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

var deviceClassToGuid = map[DeviceClass]C.GUID{
	// {745a17a0-74d3-11d0-b6fe-00a0c90f57da}
	DevHid: C.GUID{
		0x745A17A0, 0x74D3, 0x11D0,
		[8]C.uchar{0xB6, 0xFE, 0x00, 0xA0, 0xC9, 0x0F, 0x57, 0xDA},
	},

	// {36fc9e60-c465-11cf-8056-444553540000}
	DevUsbDevice: C.GUID{
		0x36FC9E60, 0xC465, 0x11CF,
		[8]C.uchar{0x80, 0x56, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00},
	},
}

var guidToInterfaceClass map[C.GUID]InterfaceClass
var guidToDeviceClass map[C.GUID]DeviceClass

func init() {
	guidToInterfaceClass = make(map[C.GUID]InterfaceClass)
	for interfaceClass, guid := range interfaceClassToGuid {
		guidToInterfaceClass[guid] = interfaceClass
	}

	guidToDeviceClass = make(map[C.GUID]DeviceClass)
	for deviceClass, guid := range deviceClassToGuid {
		guidToDeviceClass[guid] = deviceClass
	}
}
