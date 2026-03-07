//go:build windows

package dport

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const headerFutexSize = 0

const fileMapAllAccess = windows.FILE_MAP_WRITE | windows.FILE_MAP_READ

var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procOpenFileMapping = modkernel32.NewProc("OpenFileMappingW")
)

type platformHandle struct {
	hMapFile windows.Handle
}

func createShm(name string, totalSize uintptr) (unsafe.Pointer, platformHandle, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, platformHandle{}, err
	}

	h, err := windows.CreateFileMapping(
		windows.InvalidHandle,
		nil,
		windows.PAGE_READWRITE,
		0,
		uint32(totalSize),
		namePtr,
	)
	if err != nil {
		return nil, platformHandle{}, err
	}

	ptr, err := windows.MapViewOfFile(h, fileMapAllAccess, 0, 0, totalSize)
	if err != nil {
		windows.CloseHandle(h)
		return nil, platformHandle{}, err
	}

	return unsafe.Pointer(ptr), platformHandle{hMapFile: h}, nil
}

func openShm(name string) (unsafe.Pointer, platformHandle, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, platformHandle{}, err
	}

	r0, _, e1 := procOpenFileMapping.Call(
		uintptr(fileMapAllAccess),
		0, // bInheritHandle = FALSE
		uintptr(unsafe.Pointer(namePtr)),
	)
	if r0 == 0 {
		return nil, platformHandle{}, e1
	}
	h := windows.Handle(r0)

	ptr, err := windows.MapViewOfFile(h, fileMapAllAccess, 0, 0, 0)
	if err != nil {
		windows.CloseHandle(h)
		return nil, platformHandle{}, err
	}

	return unsafe.Pointer(ptr), platformHandle{hMapFile: h}, nil
}

func closeShm(basePtr unsafe.Pointer, _ uintptr, handle platformHandle, _ bool) {
	windows.UnmapViewOfFile(uintptr(basePtr))
	windows.CloseHandle(handle.hMapFile)
}
