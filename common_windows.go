//go:build windows

package hotplug

import (
	"errors"
	"fmt"
	"slices"
	"unsafe"
)

/*
	#cgo LDFLAGS: -lcfgmgr32

	// initguid.h redefines the DEFINE_GUID macro so that the Windows headers
	// instantiate their GUIDs rather than just declaring extern references.
	// It must be included in the application exactly once.
	#include <initguid.h>

	#include "common_windows.h"
*/
import "C"

// splitUTF16StringList converts a Windows REG_MULTI_SZ/ZZWSTR list of strings
// provided as a slice of WCHAR to a slice of strings each of which is a slice
// of WCHAR including one null termination.
func splitUTF16StringList(list []uint16) [][]uint16 {
	out := make([][]uint16, 10)
	tail := list
	for {
		nextNull := slices.Index(tail, 0)
		if nextNull <= 0 {
			break
		}

		head := tail[:nextNull+1]
		tail = tail[nextNull+1:]

		out = append(out, head)
	}

	return out
}

func getDevIfPropFixed[T any](
	deviceInterfaceId []uint16,
	propKey *C.DEVPROPKEY,
	expectedType C.DEVPROPTYPE,
	out *T,
) error {
	var actualType C.DEVPROPTYPE
	var size C.ULONG = (C.ULONG)(unsafe.Sizeof(*out))
	sta := C.CM_Get_Device_Interface_PropertyW(
		(*C.WCHAR)(unsafe.SliceData(deviceInterfaceId)),
		propKey,
		&actualType,
		(*C.BYTE)(unsafe.Pointer(out)),
		&size,
		0,
	)
	if sta != C.CR_SUCCESS {
		return errors.New(fmt.Sprintf(
			"failed to get property value (CONFIGRET 0x%X)",
			sta,
		))
	} else if actualType != expectedType {
		return errors.New(fmt.Sprintf(
			"property type mismatch (got 0x%X, expected 0x%X)",
			actualType,
			expectedType,
		))
	}

	return nil
}

func getDevPropFixed[T any](
	deviceInstance C.DEVINST,
	propKey *C.DEVPROPKEY,
	expectedType C.DEVPROPTYPE,
	out *T,
) error {
	var actualType C.DEVPROPTYPE
	var size C.ULONG = (C.ULONG)(unsafe.Sizeof(*out))
	sta := C.CM_Get_DevNode_PropertyW(
		deviceInstance,
		propKey,
		&actualType,
		(*C.BYTE)(unsafe.Pointer(out)),
		&size,
		0,
	)
	if sta != C.CR_SUCCESS {
		return errors.New(fmt.Sprintf(
			"failed to get property value (CONFIGRET 0x%X)",
			sta,
		))
	} else if actualType != expectedType {
		return errors.New(fmt.Sprintf(
			"property type mismatch (got 0x%X, expected 0x%X)",
			actualType,
			expectedType,
		))
	}

	return nil
}
