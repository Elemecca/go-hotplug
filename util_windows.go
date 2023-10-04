//go:build windows

package hotplug

// #include <windows.h>
import "C"

import (
	"fmt"
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
