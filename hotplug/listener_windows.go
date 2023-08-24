//go:build windows

package hotplug

import (
	"errors"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
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
	attach       bool
	deviceClass  uuid.UUID
	symbolicLink string
}

type listenerData struct {
	handle      cgo.Handle
	notifHandle C.HCMNOTIFICATION
	eventChan   chan *devInterfaceNotification
}

func (dm *Listener) enable() error {
	dm.handle = cgo.NewHandle(dm)
	dm.eventChan = make(chan *devInterfaceNotification, 10)

	go dm.devIfLoop()

	var filter C.CM_NOTIFY_FILTER
	filter.cbSize = C.sizeof_CM_NOTIFY_FILTER
	filter.FilterType = C.CM_NOTIFY_FILTER_TYPE_DEVICEINTERFACE
	// filter.u.DeviceInterface.ClassGuid
	//*((*C.GUID)(unsafe.Pointer(&filter.u[0]))) = usbDeviceClass
	filter.Flags = C.CM_NOTIFY_FILTER_FLAG_ALL_INTERFACE_CLASSES

	res := C.CM_Register_Notification(
		&filter,
		(C.PVOID)(unsafe.Pointer(&dm.handle)),
		(C.PCM_NOTIFY_CALLBACK)(C.configNotificationHandler),
		&dm.notifHandle,
	)
	if res != C.CR_SUCCESS {
		return errors.New("CM_Register_Notification failed")
	}

	log.Debug("openHotplug done")
	return nil
}

func (dm *Listener) disable() error {
	// blocks while it delivers all pending notifications
	res := C.CM_Unregister_Notification(dm.notifHandle)
	if res != C.CR_SUCCESS {
		return errors.New("CM_Register_Notification failed")
	}

	dm.handle.Delete()
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
			action == C.CM_NOTIFY_ACTION_DEVICEINTERFACEARRIVAL,
			// data.u.DeviceInterface.ClassGuid
			guidToUuid((*C.GUID)(unsafe.Pointer(&data.u[0]))),
			// data.u.DeviceInterface.SymbolicLink
			windows.UTF16PtrToString((*uint16)(unsafe.Pointer(&data.u[C.sizeof_GUID]))),
		}
	}

	return C.ERROR_SUCCESS
}

func (dm *Listener) devIfLoop() {
	for {
		notif := <-dm.eventChan
		clog := log.WithFields(log.Fields{
			"class":   notif.deviceClass,
			"symlink": notif.symbolicLink,
		})

		if notif.attach {
			clog.Debug("USB device arrived")
		} else {
			clog.Debug("USB device left")
		}

	}
}

func (l *Listener) enumerate() error {

}
