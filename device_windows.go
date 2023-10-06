//go:build windows

package hotplug

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	enumerator     string
	hardwareIds    map[string]string
	compatibleIds  map[string]string
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

func (dev *Device) getStringListProperty(key *C.DEVPROPKEY) ([]string, error) {
	buf, err := dev.getProperty(key, C.DEVPROP_TYPE_STRING_LIST)
	if err != nil {
		return nil, err
	}

	// reinterpret cast []byte to []WCHAR
	wcBuf := unsafe.Slice((*C.WCHAR)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf))

	wcharStrings := splitWcharStringList(wcBuf)
	out := make([]string, len(wcharStrings))
	for idx, wcharString := range wcharStrings {
		out[idx], err = wcharToGoString(unsafe.SliceData(wcharString))
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (dev *Device) getInt32Property(key *C.DEVPROPKEY) (int, error) {
	buf, err := dev.getProperty(key, C.DEVPROP_TYPE_INT32)
	if err != nil {
		return 0, err
	}

	return (int)(*(*C.LONG)(unsafe.Pointer(unsafe.SliceData(buf)))), nil
}

func (dev *Device) path() (string, error) {
	return wcharToGoString(unsafe.SliceData(dev.symbolicLink))
}

func (dev *Device) class() (DeviceClass, error) {
	if dev.classGuid != (C.GUID{}) {
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

var idRe = regexp.MustCompile(`^\\\\[^\\]+\\([^\\]+)(?:\\|#|$)`)
var paramRe = regexp.MustCompile(`^([A-Z]+)_([^&]+)(?:&|$)`)

func (dev *Device) getIds(key *C.DEVPROPKEY) (map[string]string, error) {
	idStrings, err := dev.getStringListProperty(key)
	if err != nil {
		return nil, err
	}

	params := make(map[string]string)
	for _, idString := range idStrings {
		idMatch := idRe.FindStringSubmatch(idString)
		if idMatch == nil {
			continue
		}

		for _, param := range paramRe.FindAllStringSubmatch(idMatch[1], -1) {
			params[param[1]] = param[2]
		}
	}

	return params, nil
}

func (dev *Device) getHardwareIds() (map[string]string, error) {
	if dev.hardwareIds == nil {
		return dev.hardwareIds, nil
	}

	hardwareIds, err := dev.getIds(&C.DEVPKEY_Device_HardwareIds)
	if err != nil {
		return nil, err
	}

	dev.hardwareIds = hardwareIds
	return hardwareIds, nil
}

func (dev *Device) getHardwareId(key string) (string, error) {
	hardwareIds, err := dev.getHardwareIds()
	if err != nil {
		return "", err
	}

	val, ok := hardwareIds[key]
	if ok {
		return val, nil
	} else {
		return "", errors.New(fmt.Sprintf("HardwareIds does not contain %s property", key))
	}
}

func (dev *Device) getHardwareIdHex(key string) (int, error) {
	str, err := dev.getHardwareId(key)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseInt(str, 16, 32)
	if err != nil {
		return 0, err
	}

	return (int)(val), nil
}

func (dev *Device) getCompatibleIds() (map[string]string, error) {
	if dev.compatibleIds == nil {
		return dev.compatibleIds, nil
	}

	compatibleIds, err := dev.getIds(&C.DEVPKEY_Device_CompatibleIds)
	if err != nil {
		return nil, err
	}

	dev.compatibleIds = compatibleIds
	return compatibleIds, nil
}

func (dev *Device) bus() (Bus, error) {
	return UnknownBus, errors.New("not implemented")
}

func (dev *Device) busNumber() (int, error) {
	return dev.getInt32Property(&C.DEVPKEY_Device_BusNumber)
}

func (dev *Device) address() (int, error) {
	return dev.getInt32Property(&C.DEVPKEY_Device_Address)
}

func (dev *Device) vendorId() (int, error) {
	return dev.getHardwareIdHex("VID")
}

func (dev *Device) productId() (int, error) {
	return dev.getHardwareIdHex("PID")
}
