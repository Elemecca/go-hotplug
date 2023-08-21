package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

/*
	#cgo LDFLAGS: -lcfgmgr32 -lpropsys -lole32
	#define WINVER 0x0602 // Windows 8
	#define UNICODE
    #include <windows.h>
    #include <cfgmgr32.h>
	#include <propsys.h>

	// these are missing from cfgmgr32.h in mingw-w64
	CMAPI CONFIGRET CM_Get_DevNode_Property_Keys(DEVINST dnDevInst, DEVPROPKEY *PropertyKeyArray, PULONG PropertyKeyCount, ULONG ulFlags );
	CMAPI CONFIGRET CM_Get_Device_Interface_Property_KeysW(LPCWSTR pszDeviceInterface, DEVPROPKEY *PropertyKeyArray, PULONG PropertyKeyCount, ULONG ulFlags);
*/
import "C"

func main() {
	GUID_DEVINTERFACE_HID := C.GUID{
		0x4D1E55B2,
		0xF16F,
		0x11CF,
		[8]C.uchar{
			0x88,
			0xCB,
			0x00,
			0x11,
			0x11,
			0x00,
			0x00,
			0x30,
		},
	}

	instId, _ := windows.UTF16PtrFromString(
		"USB\\VID_05E0&PID_1900&MI_00\\9&F8684BC&0&0000",
	)

	var deviceInstance C.DEVINST
	status := C.CM_Locate_DevNode(
		&deviceInstance,
		(*C.WCHAR)(instId),
		C.CM_LOCATE_DEVNODE_NORMAL,
	)
	if status != C.CR_SUCCESS {
		fmt.Printf(
			"failed to locate device node (CONFIGRET 0x%X)",
			status,
		)
		return
	}

	printDevNodePropertyKeys(deviceInstance)

	var ifaceListSize C.ULONG
	status = C.CM_Get_Device_Interface_List_Size(
		&ifaceListSize,
		&GUID_DEVINTERFACE_HID,
		nil,
		C.CM_GET_DEVICE_INTERFACE_LIST_ALL_DEVICES,
		//(*C.WCHAR)(instId),
		//0,
	)
	if status != C.CR_SUCCESS {
		fmt.Printf(
			"failed to get interface list size (CONFIGRET 0x%X)",
			status,
		)
		return
	}

	interfaceList := make([]uint16, ifaceListSize)
	status = C.CM_Get_Device_Interface_List(
		&GUID_DEVINTERFACE_HID,
		nil,
		//(*C.WCHAR)(instId),
		(*C.WCHAR)(unsafe.SliceData(interfaceList)),
		ifaceListSize,
		C.CM_GET_DEVICE_INTERFACE_LIST_ALL_DEVICES,
		//0,
	)
	if status != C.CR_SUCCESS {
		fmt.Printf(
			"failed to get interface list (CONFIGRET 0x%X)",
			status,
		)
		return
	}

	printDevInterfacePropertyKeys((*C.WCHAR)(unsafe.SliceData(interfaceList)))
}

func printDevNodePropertyKeys(deviceInstance C.DEVINST) {
	var propKeyCount C.ulong
	status := C.CM_Get_DevNode_Property_Keys(
		deviceInstance,
		nil,
		&propKeyCount,
		0,
	)
	if status != C.CR_BUFFER_SMALL {
		fmt.Printf(
			"failed to get property key count (CONFIGRET 0x%X)",
			status,
		)
		return
	}

	propKeys := make([]C.DEVPROPKEY, propKeyCount)
	status = C.CM_Get_DevNode_Property_Keys(
		deviceInstance,
		unsafe.SliceData(propKeys),
		&propKeyCount,
		0,
	)
	if status != C.CR_SUCCESS {
		fmt.Printf(
			"failed to get property keys (CONFIGRET 0x%X)",
			status,
		)
		return
	}

	fmt.Println("** DevNode Properties ")
	for _, key := range propKeys {
		printPropertyKey(&key)
	}
}

func printDevInterfacePropertyKeys(deviceIfaceId *C.WCHAR) {
	fmt.Println(windows.UTF16PtrToString((*uint16)(deviceIfaceId)))

	var propKeyCount C.ulong
	status := C.CM_Get_Device_Interface_Property_KeysW(
		deviceIfaceId,
		nil,
		&propKeyCount,
		0,
	)
	if status != C.CR_BUFFER_SMALL {
		fmt.Printf(
			"failed to get devif property key count (CONFIGRET 0x%X)",
			status,
		)
		return
	}

	propKeys := make([]C.DEVPROPKEY, propKeyCount)
	status = C.CM_Get_Device_Interface_Property_KeysW(
		deviceIfaceId,
		unsafe.SliceData(propKeys),
		&propKeyCount,
		0,
	)
	if status != C.CR_SUCCESS {
		fmt.Printf(
			"failed to get devif property keys (CONFIGRET 0x%X)",
			status,
		)
		return
	}

	fmt.Println("** Device Interface Properties ")
	for _, key := range propKeys {
		printPropertyKey(&key)
	}
}

func printPropertyKey(key *C.DEVPROPKEY) {
	var propName C.PWSTR
	res := C.PSGetNameFromPropertyKey(
		(*C.PROPERTYKEY)(key),
		&propName,
	)
	if res == C.S_OK {
		fmt.Println(windows.UTF16PtrToString((*uint16)(unsafe.Pointer(propName))))
		C.CoTaskMemFree((C.LPVOID)(unsafe.Pointer(propName)))
	} else {
		fmt.Printf(
			"0x%08x, 0x%04x, 0x%04x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, %d\n",
			key.fmtid.Data1,
			key.fmtid.Data2,
			key.fmtid.Data3,
			key.fmtid.Data4[0],
			key.fmtid.Data4[1],
			key.fmtid.Data4[2],
			key.fmtid.Data4[3],
			key.fmtid.Data4[4],
			key.fmtid.Data4[5],
			key.fmtid.Data4[6],
			key.fmtid.Data4[7],
			key.pid,
		)
	}
}
