//go:build linux

package dport

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const headerFutexSize = 4

type platformHandle struct {
	mmapSlice []byte
}

func createShm(name string, totalSize uintptr) (unsafe.Pointer, platformHandle, error) {
	fd, err := shmOpen(name, unix.O_CREAT|unix.O_RDWR, 0666)
	if err != nil {
		return nil, platformHandle{}, fmt.Errorf("shm_open: %w", err)
	}
	defer unix.Close(fd)

	if err := unix.Ftruncate(fd, int64(totalSize)); err != nil {
		return nil, platformHandle{}, fmt.Errorf("ftruncate: %w", err)
	}

	b, err := unix.Mmap(fd, 0, int(totalSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return nil, platformHandle{}, fmt.Errorf("mmap: %w", err)
	}

	return unsafe.Pointer(&b[0]), platformHandle{mmapSlice: b}, nil
}

func openShm(name string) (unsafe.Pointer, platformHandle, error) {
	fd, err := shmOpen(name, unix.O_RDWR, 0666)
	if err != nil {
		return nil, platformHandle{}, fmt.Errorf("shm_open: %w", err)
	}
	defer unix.Close(fd)

	hdr, err := unix.Mmap(fd, 0, int(headerSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return nil, platformHandle{}, fmt.Errorf("mmap header: %w", err)
	}

	shmSize := *(*uintptr)(unsafe.Pointer(&hdr[0]))
	unix.Munmap(hdr)

	totalSize := shmSize + uintptr(headerSize)
	b, err := unix.Mmap(fd, 0, int(totalSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return nil, platformHandle{}, fmt.Errorf("mmap full: %w", err)
	}

	return unsafe.Pointer(&b[0]), platformHandle{mmapSlice: b}, nil
}

func closeShm(_ unsafe.Pointer, _ uintptr, handle platformHandle, isCreator bool) {
	unix.Munmap(handle.mmapSlice)
}

func shmOpen(name string, oflag int, mode uint32) (int, error) {
	path := "/dev/shm/" + name
	return unix.Open(path, oflag, mode)
}
