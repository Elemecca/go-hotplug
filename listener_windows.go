//go:build windows

package hotplug

import (
	"errors"
	"runtime/cgo"
	"unsafe"
)

/*
	#cgo LDFLAGS: -lcfgmgr32
	#define WINVER 0x0602 // Windows 8
	#define UNICODE
	#include <windows.h>
	#include <cfgmgr32.h>
    #include <string.h>

	// these are missing from cfgmgr32.h in mingw-w64
	CMAPI CONFIGRET WINAPI CM_Register_Notification(PCM_NOTIFY_FILTER pFilter, PVOID pContext, PCM_NOTIFY_CALLBACK pCallback, PHCMNOTIFICATION pNotifyContext);
	CMAPI CONFIGRET WINAPI CM_Unregister_Notification(HCMNOTIFICATION NotifyContext);

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
	eventChan   chan *Device
}

func (l *Listener) init() error {
	return nil
}

func (l *Listener) listen() error {
	if l.notifHandle != nil {
		return errors.New("listener is already listening")
	}

	l.handle = cgo.NewHandle(l)
	l.eventChan = make(chan *Device, 10)

	go l.eventPump()

	var filter C.CM_NOTIFY_FILTER
	filter.cbSize = C.sizeof_CM_NOTIFY_FILTER
	filter.FilterType = C.CM_NOTIFY_FILTER_TYPE_DEVICEINTERFACE
	// filter.u.DeviceInterface.ClassGuid
	*((*C.GUID)(unsafe.Pointer(&filter.u[0]))) = deviceClassToGuid[l.class]

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
		dev := &Device{}

		// data.u.DeviceInterface.ClassGuid
		dev.classGuid = *(*C.GUID)(unsafe.Pointer(&data.u[0]))

		// data.u.DeviceInterface.SymbolicLink
		eventSymLink := (*C.WCHAR)(unsafe.Pointer(&data.u[C.sizeof_GUID]))

		// the documentation is not entirely clear on memory ownership
		// but it doesn't say the callee needs to free the event structure
		// so it probably belongs to the caller
		// copy the symbolic link string into go-managed memory
		length := C.wcslen(eventSymLink) + 1
		dev.symbolicLink = make([]C.WCHAR, length)
		C.wcsncpy(unsafe.SliceData(dev.symbolicLink), eventSymLink, length)

		// this function must return promptly to avoid holding up the system
		// so do the filtering and the user callback in a separate goroutine
		// this could still block if the channel buffer fills up
		l.eventChan <- dev
	}

	return C.ERROR_SUCCESS
}

func (l *Listener) eventPump() {
	for dev := range l.eventChan {
		l.callback(dev, true)
	}
}

func (l *Listener) enumerate() error {
	classGuid := deviceClassToGuid[l.class]
	var bufSize C.ULONG
	var buf []C.WCHAR

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

		buf = make([]C.WCHAR, bufSize)

		res = C.CM_Get_Device_Interface_ListW(
			&classGuid,
			nil,
			unsafe.SliceData(buf),
			bufSize,
			C.CM_GET_DEVICE_INTERFACE_LIST_PRESENT,
		)
		if res == C.CR_SUCCESS {
			break
		} else if res != C.CR_BUFFER_SMALL {
			return errors.New("CM_Get_Device_Interface_List failed")
		}
	}

	for _, symbolicLink := range splitWcharStringList(buf) {
		dev := &Device{}
		dev.classGuid = classGuid
		dev.symbolicLink = symbolicLink
		l.callback(dev, true)
	}

	return nil
}
