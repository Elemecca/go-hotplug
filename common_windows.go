//go:build windows

package hotplug

import (
	"slices"
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
