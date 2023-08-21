//go:build windows

package hotplug

// #include <windows.h>
import "C"

import (
	"fmt"
	"github.com/google/uuid"
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

// guidToUuid converts a Windows GUID struct to a Go UUID
func guidToUuid(guid *C.GUID) uuid.UUID {
	guidBytes := (*[C.sizeof_GUID]byte)(unsafe.Pointer(guid))
	return uuid.UUID{
		// Data1 long LE -> BE
		guidBytes[3],
		guidBytes[2],
		guidBytes[1],
		guidBytes[0],
		// Data2 short LE -> BE
		guidBytes[5],
		guidBytes[4],
		// Data3 short LE -> BE
		guidBytes[7],
		guidBytes[6],
		// Data4 char[8], already BE
		guidBytes[8],
		guidBytes[9],
		guidBytes[10],
		guidBytes[11],
		guidBytes[12],
		guidBytes[13],
		guidBytes[14],
		guidBytes[15],
	}
}
