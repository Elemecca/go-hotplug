//go:build windows

package hotplug

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"regexp"
	"strconv"
)

// #include "common_windows.h"
import "C"

type platformDeviceInterface struct {
	symbolicLink []uint16
	classGuid    C.GUID
	inArrive     bool
	listener     *Listener
}

func (devIf *DeviceInterface) onDetach(callback func()) error {
	if !devIf.inArrive {
		return errors.New("OnDetach must be called from the arrive callback")
	}

	callbacks := devIf.listener.detachCb[devIf.Path]
	if callbacks == nil {
		callbacks = make([]func(), 1)
	}

	devIf.listener.detachCb[devIf.Path] = append(callbacks, callback)
	return nil
}

type platformDevice struct {
	deviceInstance C.DEVINST
	classGuid      C.GUID
	cacheVendorId  int
	cacheProductId int
	cacheSerial    string
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

func (dev *Device) busNumber() (int, error) {
	var result uint32
	err := getDevPropFixed(
		dev.deviceInstance,
		&C.DEVPKEY_Device_BusNumber,
		C.DEVPROP_TYPE_UINT32,
		&result,
	)
	return (int)(result), err
}

func (dev *Device) address() (int, error) {
	var result uint32
	err := getDevPropFixed(
		dev.deviceInstance,
		&C.DEVPKEY_Device_Address,
		C.DEVPROP_TYPE_UINT32,
		&result,
	)
	return (int)(result), err
}

var reUsbPath = regexp.MustCompile(`^USB\\VID_([0-9A-F]{4})&PID_([0-9A-F]{4})\\(.+)$`)

func (dev *Device) parseUsbPath() error {
	match := reUsbPath.FindStringSubmatch(dev.Path)
	if match == nil {
		return errors.New("device Path does not match expected pattern")
	}

	vendorId, err := strconv.ParseInt(match[1], 16, 32)
	if err != nil {
		return err
	}

	productId, err := strconv.ParseInt(match[2], 16, 32)
	if err != nil {
		return err
	}

	dev.cacheVendorId = (int)(vendorId)
	dev.cacheProductId = (int)(productId)
	dev.cacheSerial = match[3]
	return nil
}

func (dev *Device) vendorId() (int, error) {
	if dev.Class == DevUsbDevice {
		if dev.cacheVendorId == 0 {
			err := dev.parseUsbPath()
			if err != nil {
				return 0, err
			}
		}
		return dev.cacheVendorId, nil
	} else {
		return 0, errors.New("property not supported for this DeviceClass")
	}
}

func (dev *Device) productId() (int, error) {
	if dev.Class == DevUsbDevice {
		if dev.cacheProductId == 0 {
			err := dev.parseUsbPath()
			if err != nil {
				return 0, err
			}
		}
		return dev.cacheProductId, nil
	} else {
		return 0, errors.New("property not supported for this DeviceClass")
	}
}
