//go:build windows

package hotplug

import (
	"errors"
	"golang.org/x/sys/windows"
	"runtime/cgo"
	"unsafe"
)

/*
	#include "common_windows.h"

	DWORD configNotificationHandler(
		HCMNOTIFICATION hNotify,
		PVOID context,
		CM_NOTIFY_ACTION action,
		PCM_NOTIFY_EVENT_DATA eventData,
		DWORD eventDataSize
	);
*/
import "C"

type platformListener struct {
	handle      cgo.Handle
	notifHandle C.HCMNOTIFICATION
	eventChan   chan *DeviceInterface
}

func (l *Listener) init() error {
	return nil
}

func (l *Listener) listen() error {
	if l.notifHandle != nil {
		return errors.New("listener is already listening")
	}

	l.handle = cgo.NewHandle(l)
	l.eventChan = make(chan *DeviceInterface, 10)

	go l.eventPump()

	var filter C.CM_NOTIFY_FILTER
	filter.cbSize = C.sizeof_CM_NOTIFY_FILTER
	filter.FilterType = C.CM_NOTIFY_FILTER_TYPE_DEVICEINTERFACE
	// filter.u.DeviceInterface.ClassGuid
	*((*C.GUID)(unsafe.Pointer(&filter.u[0]))) = interfaceClassToGuid[l.class]

	res := C.CM_Register_Notification(
		&filter,
		(C.PVOID)(unsafe.Pointer(&l.handle)),
		(C.PCM_NOTIFY_CALLBACK)(C.configNotificationHandler),
		&l.notifHandle,
	)
	if res != C.CR_SUCCESS {
		return errors.New("CM_Register_Notification failed")
	}

	return nil
}

func (l *Listener) stop() (err error) {
	if l.notifHandle == nil {
		return errors.New("listener is not listening")
	}

	// blocks while it delivers all pending notifications
	res := C.CM_Unregister_Notification(l.notifHandle)
	if res != C.CR_SUCCESS {
		err = errors.New("CM_Unregister_Notification failed")
	}

	l.notifHandle = nil
	close(l.eventChan)
	l.handle.Delete()
	return
}

//export configNotificationHandler
func configNotificationHandler(
	hNotify C.HCMNOTIFICATION,
	context unsafe.Pointer,
	action C.CM_NOTIFY_ACTION,
	data C.PCM_NOTIFY_EVENT_DATA,
	eventDataSize C.DWORD,
) C.DWORD {
	l := (*(*cgo.Handle)(context)).Value().(*Listener)

	if action == C.CM_NOTIFY_ACTION_DEVICEINTERFACEARRIVAL {
		devIf := &DeviceInterface{}

		// data.u.DeviceInterface.ClassGuid
		devIf.classGuid = *(*C.GUID)(unsafe.Pointer(&data.u[0]))

		// data.u.DeviceInterface.SymbolicLink
		eventSymLink := (*C.WCHAR)(unsafe.Pointer(&data.u[C.sizeof_GUID]))

		// the documentation is not entirely clear on memory ownership
		// but it doesn't say the callee needs to free the event structure
		// so it probably belongs to the caller
		// copy the symbolic link string into go-managed memory
		length := C.wcslen(eventSymLink) + 1
		devIf.symbolicLink = make([]uint16, length)
		C.wcsncpy((*C.WCHAR)(unsafe.SliceData(devIf.symbolicLink)), eventSymLink, length)

		// this function must return promptly to avoid holding up the system
		// so do the filtering and the user callback in a separate goroutine
		// this could still block if the channel buffer fills up
		l.eventChan <- devIf
	}

	return C.ERROR_SUCCESS
}

func (l *Listener) eventPump() {
	for devIf := range l.eventChan {
		l.handleArrive(devIf)
	}
}

func (l *Listener) enumerate() error {
	classGuid := interfaceClassToGuid[l.class]
	var bufSize C.ULONG
	var buf []uint16

	// the list can change between the calls to _List_Size and _List
	// if that happens _List returns CR_BUFFER_SMALL and we try again
	for {
		res := C.CM_Get_Device_Interface_List_SizeW(
			&bufSize,
			&classGuid,
			nil,
			C.CM_GET_DEVICE_INTERFACE_LIST_PRESENT,
		)
		if res != C.CR_SUCCESS {
			return errors.New("CM_Get_Device_Interface_List_Size failed")
		}

		buf = make([]uint16, bufSize)
		res = C.CM_Get_Device_Interface_ListW(
			&classGuid,
			nil,
			(*C.WCHAR)(unsafe.SliceData(buf)),
			bufSize,
			C.CM_GET_DEVICE_INTERFACE_LIST_PRESENT,
		)
		if res == C.CR_SUCCESS {
			break
		} else if res != C.CR_BUFFER_SMALL {
			return errors.New("CM_Get_Device_Interface_List failed")
		}
	}

	for _, symbolicLink := range splitUTF16StringList(buf) {
		devIf := &DeviceInterface{}
		devIf.classGuid = classGuid
		devIf.symbolicLink = symbolicLink
		l.handleArrive(devIf)
	}

	return nil
}

func (l *Listener) handleArrive(devIf *DeviceInterface) {
	devIf.Path = windows.UTF16ToString(devIf.symbolicLink)
	devIf.Class = guidToInterfaceClass[devIf.classGuid]
	devIf.Device = &Device{}

	var propType C.DEVPROPTYPE
	var devInstanceId [C.MAX_DEVICE_ID_LEN + 1]uint16
	var size C.ULONG

	size = (C.ULONG)(unsafe.Sizeof(devInstanceId))
	status := C.CM_Get_Device_Interface_PropertyW(
		(*C.WCHAR)(unsafe.SliceData(devIf.symbolicLink)),
		&C.DEVPKEY_Device_InstanceId,
		&propType,
		(C.PBYTE)(unsafe.Pointer(&devInstanceId[0])),
		&size,
		0,
	)
	if status != C.CR_SUCCESS || propType != C.DEVPROP_TYPE_STRING {
		return
	}

	status = C.CM_Locate_DevNodeW(
		&devIf.Device.deviceInstance,
		(*C.WCHAR)(&devInstanceId[0]),
		C.CM_LOCATE_DEVNODE_NORMAL,
	)
	if status != C.CR_SUCCESS {
		return
	}

	size = (C.ULONG)(unsafe.Sizeof(devIf.Device.classGuid))
	status = C.CM_Get_DevNode_PropertyW(
		devIf.Device.deviceInstance,
		&C.DEVPKEY_Device_ClassGuid,
		&propType,
		(C.PBYTE)(unsafe.Pointer(&devIf.Device.classGuid)),
		&size,
		0,
	)
	if status != C.CR_SUCCESS || propType != C.DEVPROP_TYPE_GUID {
		return
	}

	devIf.Device.Path = windows.UTF16ToString(devInstanceId[:])
	devIf.Device.Class = guidToDeviceClass[devIf.Device.classGuid]

	l.callback(devIf)
}
