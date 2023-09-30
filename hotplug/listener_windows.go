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

type devInterfaceNotification struct {
	device *Device
	attach bool
}

type listenerData struct {
	handle       cgo.Handle
	notifHandles []C.HCMNOTIFICATION
	eventChan    chan *devInterfaceNotification
}

func (l *Listener) enable() error {
	l.handle = cgo.NewHandle(l)
	l.eventChan = make(chan *devInterfaceNotification, 10)

	go l.devIfLoop()

	for _, devType := range l.devTypes {
		err := l.enableType(devType)
		if err != nil {
			_ = l.disable()
			return err
		}
	}

	return nil
}

func (l *Listener) enableType(devType DeviceClass) error {
	l.notifHandles = append(l.notifHandles, 0)

	var filter C.CM_NOTIFY_FILTER
	filter.cbSize = C.sizeof_CM_NOTIFY_FILTER
	filter.FilterType = C.CM_NOTIFY_FILTER_TYPE_DEVICEINTERFACE
	// filter.u.DeviceInterface.ClassGuid
	*((*C.GUID)(unsafe.Pointer(&filter.u[0]))) = deviceClassToGuid[devType]
	//filter.Flags = C.CM_NOTIFY_FILTER_FLAG_ALL_INTERFACE_CLASSES

	res := C.CM_Register_Notification(
		&filter,
		(C.PVOID)(unsafe.Pointer(&l.handle)),
		(C.PCM_NOTIFY_CALLBACK)(C.configNotificationHandler),
		&l.notifHandles[len(l.notifHandles)-1],
	)
	if res != C.CR_SUCCESS {
		return errors.New("CM_Register_Notification failed")
	}

	return nil
}

func (l *Listener) disable() error {
	for _, notifHandle := range l.notifHandles {
		// blocks while it delivers all pending notifications
		res := C.CM_Unregister_Notification(notifHandle)
		if res != C.CR_SUCCESS {
			return errors.New("CM_Unregister_Notification failed")
		}
	}

	l.handle.Delete()
	return nil
}

//export configNotificationHandler
func configNotificationHandler(
	hNotify C.HCMNOTIFICATION,
	context unsafe.Pointer,
	action C.CM_NOTIFY_ACTION,
	data C.PCM_NOTIFY_EVENT_DATA,
	eventDataSize C.DWORD,
) C.DWORD {
	dm := (*(*cgo.Handle)(context)).Value().(*Listener)

	isDevIface :=
		action == C.CM_NOTIFY_ACTION_DEVICEINTERFACEARRIVAL ||
			action == C.CM_NOTIFY_ACTION_DEVICEINTERFACEREMOVAL
	if isDevIface {
		dm.eventChan <- &devInterfaceNotification{
			&Device{
				// data.u.DeviceInterface.SymbolicLink
				symbolicLink: (*C.GUID)(unsafe.Pointer(&data.u[C.sizeof_GUID])),
			},
			action == C.CM_NOTIFY_ACTION_DEVICEINTERFACEARRIVAL,
		}
		// data.u.DeviceInterface.ClassGuid
		guidToUuid((*C.GUID)(unsafe.Pointer(&data.u[0]))),
	}

	return C.ERROR_SUCCESS
}

func (l *Listener) devIfLoop() {
	for {
		notif := <-l.eventChan

	}
}

func (l *Listener) enumerate() error {
	return nil
}
