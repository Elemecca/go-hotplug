//go:build windows

package hotplug

/*
	#define WINVER 0x0602 // Windows 8
	#define UNICODE
	#include <windows.h>
*/
import "C"

import (
	"fmt"
	"slices"
	"unsafe"
)

// wcharToGoString converts a Windows wchar_t* null-terminated UTF-16 string
// to a Go UTF-8 string.
func wcharToGoString(in *C.WCHAR) (string, error) {
	size := C.WideCharToMultiByte(C.CP_UTF8, 0, in, -1, nil, 0, nil, nil)
	if size == 0 {
		panic(fmt.Sprintf("WideCharToMultiByte failed with %d", C.GetLastError()))
	}

	bufSlice := make([]byte, size)
	buf := unsafe.SliceData(bufSlice)
	cbuf := (*C.char)(unsafe.Pointer(buf))

	res := C.WideCharToMultiByte(C.CP_UTF8, 0, in, -1, cbuf, size, nil, nil)
	if res == 0 {
		panic(fmt.Sprintf("WideCharToMultiByte failed with %d", C.GetLastError()))
	}

	// size includes the null termination but Go uses Pascal strings
	return unsafe.String(buf, size-1), nil
}

// splitWcharStringList converts a Windows REG_MULTI_SZ/ZZWSTR list of strings
// provided as a slice of WCHAR to a slice of strings each of which is a slice
// of WCHAR including one null termination.
func splitWcharStringList(list []C.WCHAR) [][]C.WCHAR {
	out := make([][]C.WCHAR, 10)
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
