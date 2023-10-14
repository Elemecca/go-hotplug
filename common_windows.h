//go:build windows
// cgo flags for this file are in common_windows.go

#define WINVER 0x0602 // Windows 8
#define UNICODE

#include <windows.h>
#include <cfgmgr32.h>
#include <devpkey.h>
#include <string.h>

// these are missing from cfgmgr32.h in mingw-w64
CMAPI CONFIGRET WINAPI CM_Register_Notification(PCM_NOTIFY_FILTER pFilter, PVOID pContext, PCM_NOTIFY_CALLBACK pCallback, PHCMNOTIFICATION pNotifyContext);
CMAPI CONFIGRET WINAPI CM_Unregister_Notification(HCMNOTIFICATION NotifyContext);
CMAPI CONFIGRET WINAPI CM_Get_Device_Interface_PropertyW(LPCWSTR pszDeviceInterface, const DEVPROPKEY *PropertyKey, DEVPROPTYPE *PropertyType, PBYTE PropertyBuffer, PULONG PropertyBufferSize, ULONG ulFlags);
