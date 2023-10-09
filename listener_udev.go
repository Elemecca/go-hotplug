//go:build linux

package hotplug

import (
	"errors"
	"golang.org/x/sys/unix"
	"runtime"
	"syscall"
)

/*
	#cgo pkg-config: libudev
	#include <libudev.h>
	#include <string.h>
*/
import "C"

var ADD = C.CString("add")
var DEVTYPE = C.CString("DEVTYPE")
var REMOVE = C.CString("remove")
var UDEV = C.CString("udev")

type platformListener struct {
	udev      *C.struct_udev
	monitor   *C.struct_udev_monitor
	closeChan chan interface{}
	closePipe []int
	deviceFd  int
}

func (l *Listener) init() error {
	l.udev = C.udev_new()
	if l.udev == nil {
		return errors.New("failed to create udev context")
	}

	runtime.SetFinalizer(l, freeListener)

	return nil
}

func freeListener(l *Listener) {
	C.udev_unref(l.udev)
	l.udev = nil
}

func (l *Listener) listen() (err error) {
	var flags int

	if l.monitor != nil {
		return errors.New("listener is already listening")
	}

	l.monitor = C.udev_monitor_new_from_netlink(l.udev, UDEV)
	if l.monitor == nil {
		return errors.New("failed to create udev monitor")
	}

	res := C.udev_monitor_filter_add_match_subsystem_devtype(
		l.monitor,
		deviceClassToSubsystem[l.class],
		deviceClassToDevtype[l.class],
	)
	if res < 0 {
		err = errors.New("failed to add udev filter")
		goto fail
	}

	res = C.udev_monitor_enable_receiving(l.monitor)
	if res < 0 {
		err = errors.New("failed to enable udev monitor")
		goto fail
	}

	l.deviceFd = (int)(C.udev_monitor_get_fd(l.monitor))
	if l.deviceFd < 0 {
		err = errors.New("failed to get udev monitor fd")
		goto fail
	}

	// ensure the file descriptor is close-on-exec
	flags, err = unix.FcntlInt((uintptr)(l.deviceFd), unix.F_GETFD, 0)
	if err != nil {
		goto fail
	}
	if flags&unix.FD_CLOEXEC != 0 {
		_, err = unix.FcntlInt((uintptr)(l.deviceFd), unix.F_SETFD, flags|unix.FD_CLOEXEC)
		if err != nil {
			goto fail
		}
	}

	// ensure the file descriptor is non-blocking
	// some older versions of udev are not by default
	flags, err = unix.FcntlInt((uintptr)(l.deviceFd), unix.F_GETFL, 0)
	if err != nil {
		goto fail
	}
	if flags&unix.O_NONBLOCK == 0 {
		_, err = unix.FcntlInt((uintptr)(l.deviceFd), unix.F_SETFL, flags|unix.O_NONBLOCK)
		if err != nil {
			goto fail
		}
	}

	l.closePipe = make([]int, 2)
	err = unix.Pipe(l.closePipe)
	if err != nil {
		goto fail
	}

	l.closeChan = make(chan interface{})

	go l.eventPump()
	return nil

fail:
	C.udev_monitor_unref(l.monitor)
	l.monitor = nil
	l.deviceFd = -1
	return
}

func (l *Listener) stop() error {
	if l.monitor == nil {
		return errors.New("listener is not listening")
	}

	// signal the eventPump thread to exit
	err := unix.Close(l.closePipe[1])
	if err != nil {
		return err
	}

	// wait for the eventPump thread to exit
	<-l.closeChan

	l.closeChan = nil
	l.closePipe = nil

	C.udev_monitor_unref(l.monitor)
	l.monitor = nil

	return nil
}

func (l *Listener) eventPump() {
	fds := []unix.PollFd{
		{Fd: (int32)(l.closePipe[0]), Events: unix.POLLHUP},
		{Fd: (int32)(l.deviceFd), Events: unix.POLLIN},
	}

	for {
		_, err := unix.Poll(fds, -1)
		if err != nil {
			if err.(syscall.Errno).Is(syscall.EINTR) {
				continue
			} else {
				break
			}
		}

		if fds[0].Revents != 0 {
			break
		}

		if fds[1].Revents != 0 {
			dev := C.udev_monitor_receive_device(l.monitor)
			if dev == nil {
				continue
			}

			action := C.udev_device_get_action(dev)
			if action == nil {
				continue
			}

			var present bool
			if C.strcmp(action, ADD) == 0 {
				present = true
			} else if C.strcmp(action, REMOVE) == 0 {
				present = false
			} else {
				continue
			}

			l.callback(newDevice(l, dev), present)
		}
	}

	close(l.closeChan)
}

func (l *Listener) enumerate() error {
	enumerator := C.udev_enumerate_new(l.udev)
	if nil == enumerator {
		return errors.New("failed to create udev enumerator")
	}
	defer C.udev_enumerate_unref(enumerator)

	res := C.udev_enumerate_add_match_subsystem(
		enumerator,
		deviceClassToSubsystem[l.class],
	)
	if res < 0 {
		return errors.New("failed to add udev subsystem filter")
	}

	devtype := deviceClassToDevtype[l.class]
	if devtype != nil {
		res = C.udev_enumerate_add_match_property(enumerator, DEVTYPE, devtype)
		if res < 0 {
			return errors.New("failed to add udev devtype filter")
		}
	}

	res = C.udev_enumerate_scan_devices(enumerator)
	if res < 0 {
		return errors.New("failed to perform udev enumeration")
	}

	entry := C.udev_enumerate_get_list_entry(enumerator)
	if entry == nil {
		return errors.New("failed to get udev device list")
	}

	for {
		path := C.udev_list_entry_get_name(entry)
		if path == nil {
			continue
		}

		dev := C.udev_device_new_from_syspath(l.udev, path)
		if dev == nil {
			continue
		}

		l.callback(newDevice(l, dev), true)

		entry = C.udev_list_entry_get_next(entry)
		if entry == nil {
			break
		}
	}

	return nil
}
