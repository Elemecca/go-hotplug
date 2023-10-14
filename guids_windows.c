//go:build windows

#define WINVER 0x0602 // Windows 8
#define UNICODE
#include <windows.h>

// including initguid makes Windows headers actually define their GUIDS
// rather than just declaring them extern
#include <initguid.h>
#include <devpkey.h>
