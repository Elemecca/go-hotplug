//go:build windows

package hotplug

import (
	"errors"
	"io"
	"runtime/cgo"
	"slices"
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

type listener struct {
	config       Config
	handle       cgo.Handle
	notifHandles []C.HCMNOTIFICATION
	eventChan    chan *Device
}

func listen(config Config) (io.Closer, error) {
	l := &listener{
		config: config,
	}
	l.handle = cgo.NewHandle(l)
	l.eventChan = make(chan *Device, 10)

	go l.eventPump()

	for _, devType := range l.config.OnlyClasses {
		err := l.enableType(devType)
		if err != nil {
			_ = l.Close()
			return nil, err
		}
	}

	return l, nil
}

func (l *listener) enableType(devType DeviceClass) error {
	l.notifHandles = append(l.notifHandles, nil)

	var filter C.CM_NOTIFY_FILTER
	filter.cbSize = C.sizeof_CM_NOTIFY_FILTER
	filter.FilterType = C.CM_NOTIFY_FILTER_TYPE_DEVICEINTERFACE
	// filter.u.DeviceInterface.ClassGuid
	*((*C.GUID)(unsafe.Pointer(&filter.u[0]))) = deviceClassToGuid[devType]

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

func (l *listener) Close() (err error) {
	for _, notifHandle := range l.notifHandles {
		// blocks while it delivers all pending notifications
		res := C.CM_Unregister_Notification(notifHandle)
		if res != C.CR_SUCCESS && err == nil {
			err = errors.New("CM_Unregister_Notification failed")
		}
	}

	close(l.eventChan)
	l.handle.Delete()
	return
}

func (c *Config) handleArrive(device *Device) {
	if len(c.OnlyBusses) == 0 {
		c.ArriveCallback(device)
	} else {
		bus, err := device.Bus()
		if err == nil && slices.Contains(c.OnlyBusses, bus) {
			c.ArriveCallback(device)
		}
	}
}

//export configNotificationHandler
func configNotificationHandler(
	hNotify C.HCMNOTIFICATION,
	context unsafe.Pointer,
	action C.CM_NOTIFY_ACTION,
	data C.PCM_NOTIFY_EVENT_DATA,
	eventDataSize C.DWORD,
) C.DWORD {
	l := (*(*cgo.Handle)(context)).Value().(*listener)

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

func (l *listener) eventPump() {
	for dev := range l.eventChan {
		l.config.handleArrive(dev)
	}
}

func enumerate(config Config) error {
	for _, class := range config.OnlyClasses {
		err := enumerateClass(config, class)
		if err != nil {
			return err
		}
	}
	return nil
}

func enumerateClass(config Config, class DeviceClass) error {
	classGuid := deviceClassToGuid[class]
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

	// the result is a list of concatenated null-terminated WCHAR strings
	// the list is terminated with an empty string
	tail := buf
	for {
		length := C.wcsnlen(unsafe.SliceData(tail), (C.size_t)(len(tail)))
		if length == 0 {
			break
		}

		head := tail[:length]
		tail = tail[length+1:]

		dev := &Device{}
		dev.classGuid = classGuid
		dev.symbolicLink = head
		config.handleArrive(dev)
	}

	return nil
}
