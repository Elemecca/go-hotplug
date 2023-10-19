//go:build windows

package hotplug

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"regexp"
	"strconv"
	"unsafe"
)

// #include "common_windows.h"
import "C"

type platformDeviceInterface struct {
	symbolicLink []uint16
	classGuid    C.GUID
}

func (devIf *DeviceInterface) onDetach(callback func()) error {
	return errors.New("not implemented")
}

type platformDevice struct {
	deviceInstance C.DEVINST
	classGuid      C.GUID
	enumerator     string
	hardwareIds    map[string]string
	compatibleIds  map[string]string
}

func (dev *Device) getProperty(
	key *C.DEVPROPKEY,
	expectedType C.DEVPROPTYPE,
) ([]byte, error) {
	var propType C.DEVPROPTYPE
	var size C.ULONG

	sta := C.CM_Get_DevNode_PropertyW(
		dev.deviceInstance,
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
		dev.deviceInstance,
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

	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(unsafe.SliceData(buf)))), nil
}

func (dev *Device) getStringListProperty(key *C.DEVPROPKEY) ([]string, error) {
	buf, err := dev.getProperty(key, C.DEVPROP_TYPE_STRING_LIST)
	if err != nil {
		return nil, err
	}

	// reinterpret cast []byte to []uint16
	wcBuf := unsafe.Slice((*uint16)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)/2)

	wcharStrings := splitUTF16StringList(wcBuf)
	out := make([]string, len(wcharStrings))
	for idx, wcharString := range wcharStrings {
		out[idx] = windows.UTF16ToString(wcharString)
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

func (dev *Device) parent() (*Device, error) {
	return nil, errors.New("not implemented")
}

func (dev *Device) up(class DeviceClass) (*Device, error) {
	devInst := dev.deviceInstance
	targetClassGuid, haveClassGuid := deviceClassToGuid[class]
	if !haveClassGuid {
		return nil, errors.New("not supported for that DeviceClass")
	}

	for {
		var parentInst C.DEVINST
		sta := C.CM_Get_Parent(&parentInst, devInst, 0)
		if sta != C.CR_SUCCESS {
			return nil, errors.New(fmt.Sprintf(
				"failed to get parent device (CONFIGRET 0x%X)",
				sta,
			))
		}
		devInst = parentInst

		var classGuid C.GUID
		err := getDevPropFixed(
			devInst,
			&C.DEVPKEY_Device_ClassGuid,
			C.DEVPROP_TYPE_GUID,
			&classGuid,
		)
		if err != nil {
			return nil, err
		}

		if classGuid == targetClassGuid {
			break
		}
	}

	var devInstanceId [C.MAX_DEVICE_ID_LEN + 1]uint16
	err := getDevPropFixed(
		devInst,
		&C.DEVPKEY_Device_InstanceId,
		C.DEVPROP_TYPE_STRING,
		&devInstanceId,
	)
	if err != nil {
		return nil, err
	}

	parent := &Device{}
	parent.Class = class
	parent.classGuid = targetClassGuid
	parent.deviceInstance = devInst
	parent.Path = windows.UTF16ToString(devInstanceId[:])

	return parent, nil
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
