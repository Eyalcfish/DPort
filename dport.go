package dport

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// headerSize matches the C packed DConnectionHeader, padded for alignment.
// Layout: size_t | char | uchar | uchar | [int futex_flag on Linux] | padding...
const headerSize = 64

const sizeOfSizeT = unsafe.Sizeof(uint64(0))

const (
	offShmSize         = 0
	offConnType        = 8
	offSenderFlagOff   = 16
	offReceiverFlagOff = 24
	offClientsCount    = 32
	offMessageSize     = 40
)

type DMessage struct {
	Size uintptr
	Data []byte
}

type DConnection struct {
	portName        string
	basePtr         unsafe.Pointer
	dataPtr         unsafe.Pointer
	shmSize         uintptr
	connectionType  byte
	identifier      uint64 // 1 = server (creator), 0 = client
	handle          platformHandle
	lastReadFlagOff uintptr
	mu              sync.Mutex
}

func Create(portName string, shmSize uintptr) (*DConnection, error) {

	totalSize := uintptr(headerSize) + shmSize

	basePtr, handle, err := createShm(portName, totalSize)
	if err != nil {
		return nil, err
	}

	conn := &DConnection{
		portName:       portName,
		basePtr:        basePtr,
		shmSize:        shmSize,
		connectionType: 2,
		identifier:     1,
		handle:         handle,
	}

	*(*uintptr)(basePtr) = shmSize
	*(*byte)(unsafe.Add(basePtr, offConnType)) = conn.connectionType
	(*atomic.Uint64)(unsafe.Pointer(uintptr(basePtr) + offReceiverFlagOff)).Store(0)
	(*atomic.Uint64)(unsafe.Pointer(uintptr(basePtr) + offSenderFlagOff)).Store(0)
	(*atomic.Uint64)(unsafe.Pointer(uintptr(basePtr) + offClientsCount)).Store(1)

	conn.dataPtr = unsafe.Add(basePtr, headerSize)
	return conn, nil
}

func Connect(portName string) (*DConnection, error) {
	basePtr, handle, err := openShm(portName)
	if err != nil {
		return nil, err
	}

	(*atomic.Uint64)(unsafe.Pointer(uintptr(basePtr) + offClientsCount)).Add(1)

	conn := &DConnection{
		portName:       portName,
		basePtr:        basePtr,
		shmSize:        *(*uintptr)(basePtr),
		connectionType: *(*byte)(unsafe.Add(basePtr, offConnType)),
		identifier:     (*atomic.Uint64)(unsafe.Pointer(uintptr(basePtr) + offClientsCount)).Load(),
		handle:         handle,
	}

	conn.dataPtr = unsafe.Add(basePtr, headerSize)
	return conn, nil
}

func (c *DConnection) Close() {
	closeShm(c.basePtr, uintptr(headerSize)+c.shmSize, c.handle, c.identifier == 1)
}

func (c *DConnection) Write(msg *DMessage, target uint64) error {
	if msg.Size > c.shmSize {
		return errors.New("dport: message size exceeds shared memory capacity")
	}

	for {
		if atomic.CompareAndSwapUint64((*uint64)(unsafe.Add(c.basePtr, offReceiverFlagOff)), 0, target) {
			break
		}
		runtime.Gosched()
	}

	*(*uintptr)(unsafe.Add(c.basePtr, offMessageSize)) = msg.Size

	copy(
		unsafe.Slice((*byte)(c.dataPtr), msg.Size),
		msg.Data,
	)

	atomic.StoreUint64((*uint64)(unsafe.Add(c.basePtr, offSenderFlagOff)), c.identifier)

	return nil
}

func (c *DConnection) Read() DMessage {
	for {
		if atomic.LoadUint64((*uint64)(unsafe.Add(c.basePtr, offSenderFlagOff))) != 0 && atomic.LoadUint64((*uint64)(unsafe.Add(c.basePtr, offReceiverFlagOff))) == c.identifier {
			atomic.StoreUint64((*uint64)(unsafe.Add(c.basePtr, offSenderFlagOff)), 0)
			break
		}

		runtime.Gosched()
	}

	size := atomic.LoadUint64((*uint64)(unsafe.Add(c.basePtr, offMessageSize)))
	data := make([]byte, size)
	copy(
		data,
		unsafe.Slice((*byte)(c.dataPtr), size),
	)

	atomic.StoreUint64((*uint64)(unsafe.Add(c.basePtr, offReceiverFlagOff)), 0)

	return DMessage{Size: uintptr(size), Data: data}
}

func (c *DConnection) WriteBytes(bytes []byte, target uint64) error {
	return c.Write(&DMessage{Data: bytes, Size: uintptr(len(bytes))}, target)
}
