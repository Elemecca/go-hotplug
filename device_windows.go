//go:build windows

package hotplug

import (
	"errors"
	"fmt"
	"unsafe"
)

/*
	#cgo LDFLAGS: -lcfgmgr32
	#define WINVER 0x0602 // Windows 8
	#define UNICODE
    #include <windows.h>
    #include <cfgmgr32.h>
	#include <devpkey.h>

	// these are missing from cfgmgr32.h in mingw-w64
	CMAPI CONFIGRET CM_Get_Device_Interface_PropertyW(LPCWSTR pszDeviceInterface, const DEVPROPKEY *PropertyKey, DEVPROPTYPE *PropertyType, PBYTE PropertyBuffer, PULONG PropertyBufferSize, ULONG ulFlags);
*/
import "C"

type platformDevice struct {
	symbolicLink   []C.WCHAR
	deviceInstance C.DEVINST
	classGuid      C.GUID
}

func (dev *Device) devInst() (C.DEVINST, error) {
	if dev.deviceInstance != 0 {
		return dev.deviceInstance, nil
	}

	var propType C.DEVPROPTYPE
	var devInstanceId [C.MAX_DEVICE_ID_LEN + 1]C.WCHAR
	var size C.ULONG = (C.ULONG)(len(devInstanceId) * C.sizeof_WCHAR)

	status := C.CM_Get_Device_Interface_PropertyW(
		unsafe.SliceData(dev.symbolicLink),
		&C.DEVPKEY_Device_InstanceId,
		&propType,
		(C.PBYTE)(unsafe.Pointer(&devInstanceId[0])),
		&size,
		0,
	)
	if status != C.CR_SUCCESS {
		return 0, errors.New(fmt.Sprintf(
			"failed to get device instance ID (CONFIGRET 0x%X)",
			status,
		))
	}

	status = C.CM_Locate_DevNodeW(
		&dev.deviceInstance,
		&devInstanceId[0],
		C.CM_LOCATE_DEVNODE_NORMAL,
	)
	if status != C.CR_SUCCESS {
		return 0, errors.New(fmt.Sprintf(
			"failed to locate device node (CONFIGRET 0x%X)",
			status,
		))
	}

	return dev.deviceInstance, nil
}

func (dev *Device) getProperty(
	key *C.DEVPROPKEY,
	expectedType C.DEVPROPTYPE,
) ([]byte, error) {
	devInst, err := dev.devInst()
	if err != nil {
		return nil, err
	}

	var propType C.DEVPROPTYPE
	var size C.ULONG

	sta := C.CM_Get_DevNode_PropertyW(
		devInst,
		key,
		&propType,
		nil,
		&size,
		0,
	)
	if sta != C.CR_BUFFER_SMALL {
		return nil, errors.New(fmt.Sprintf(
			"failed to get property size (CONFIGRET 0x%X)",
			sta,
		))
	}

	if propType != expectedType {
		return nil, errors.New(fmt.Sprintf(
			"property type mismatch (got 0x%X, expected 0x%X)",
			propType,
			expectedType,
		))
	}

	buf := make([]byte, size)

	sta = C.CM_Get_DevNode_PropertyW(
		devInst,
		key,
		&propType,
		(C.PBYTE)(unsafe.Pointer(unsafe.SliceData(buf))),
		&size,
		0,
	)
	if sta != C.CR_SUCCESS {
		return nil, errors.New(fmt.Sprintf(
			"failed to get property value (CONFIGRET 0x%X)",
			sta,
		))
	}

	return buf, nil
}

func (dev *Device) getStringProperty(key *C.DEVPROPKEY) (string, error) {
	buf, err := dev.getProperty(key, C.DEVPROP_TYPE_STRING)
	if err != nil {
		return "", err
	}

	return wcharToGoString((*C.WCHAR)(unsafe.Pointer(unsafe.SliceData(buf))))
}

func (dev *Device) path() (string, error) {
	return wcharToGoString(unsafe.SliceData(dev.symbolicLink))
}

func (dev *Device) class() (DeviceClass, error) {
	if dev.classGuid != 0 {
		class, ok := guidToDeviceClass[dev.classGuid]
		if ok {
			return class, nil
		} else {
			return UnknownClass, errors.New("unrecognized device interface class GUID")
		}
	} else {
		return UnknownClass, errors.New("this node is not a device interface")
	}
}

func (dev *Device) bus() (Bus, error) {

}

func (dev *Device) vendorId() (string, error) {

}

func (dev *Device) productId() (string, error) {

}

func (dev *Device) serialNumber() (string, error) {

}
