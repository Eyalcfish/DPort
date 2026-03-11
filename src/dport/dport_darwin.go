//go:build darwin

package dport

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const headerFutexSize = 0

type platformHandle struct {
	mmapSlice []byte
	shmName   string
}

const (
	sysShmOpen   = 266
	sysShmUnlink = 267
)

func shmOpen(name string, oflag int, mode int) (int, error) {
	nameBytes, err := unix.BytePtrFromString(name)
	if err != nil {
		return -1, err
	}
	fd, _, errno := unix.Syscall(sysShmOpen,
		uintptr(unsafe.Pointer(nameBytes)),
		uintptr(oflag),
		uintptr(mode),
	)
	if errno != 0 {
		return -1, fmt.Errorf("shm_open %q: %w", name, errno)
	}
	return int(fd), nil
}

func shmUnlink(name string) {
	nameBytes, err := unix.BytePtrFromString(name)
	if err != nil {
		fmt.Printf("Error string %s: %v\n", name, err)
		return
	}
	_, _, errno := unix.Syscall(sysShmUnlink,
		uintptr(unsafe.Pointer(nameBytes)),
		0, 0,
	)
	if errno != 0 {
		fmt.Printf("Error unlinking %s: errno %d\n", name, errno)
	} else {
		fmt.Printf("Successfully unlinked %s\n", name)
	}
}

func createShm(name string, totalSize uintptr) (unsafe.Pointer, platformHandle, error) {
	shmName := "/" + name

	fd, err := shmOpen(shmName, unix.O_CREAT|unix.O_RDWR, 0666)
	if err != nil {
		return nil, platformHandle{}, fmt.Errorf("shm_open: %w", err)
	}

	if err := unix.Ftruncate(fd, int64(totalSize)); err != nil {
		unix.Close(fd)
		if err == unix.EINVAL {
			fmt.Printf("Stale shared memory detected for %s. Unlinking and retrying...\n", shmName)
			shmUnlink(shmName)

			fd, err = shmOpen(shmName, unix.O_CREAT|unix.O_RDWR, 0666)
			if err != nil {
				return nil, platformHandle{}, fmt.Errorf("shm_open (retry): %w", err)
			}

			if err := unix.Ftruncate(fd, int64(totalSize)); err != nil {
				unix.Close(fd)
				return nil, platformHandle{}, fmt.Errorf("ftruncate (retry): %w", err)
			}
		} else {
			return nil, platformHandle{}, fmt.Errorf("ftruncate: %w", err)
		}
	}

	b, err := unix.Mmap(fd, 0, int(totalSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		unix.Close(fd)
		return nil, platformHandle{}, fmt.Errorf("mmap: %w", err)
	}

	unix.Close(fd)

	return unsafe.Pointer(&b[0]), platformHandle{mmapSlice: b, shmName: shmName}, nil
}

func openShm(name string) (unsafe.Pointer, platformHandle, error) {
	shmName := "/" + name
	fd, err := shmOpen(shmName, unix.O_RDWR, 0666)
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

	return unsafe.Pointer(&b[0]), platformHandle{mmapSlice: b, shmName: shmName}, nil
}

func closeShm(_ unsafe.Pointer, _ uintptr, handle platformHandle, isCreator bool) {
	unix.Munmap(handle.mmapSlice)
	if isCreator {
		shmUnlink(handle.shmName)
	}
}
