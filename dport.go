package dport

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const sizeOfSizeT = unsafe.Sizeof(uintptr(0))

// headerSize matches the C packed DConnectionHeader, padded for alignment.
// Layout: size_t | char | uchar | uchar | [int futex_flag on Linux] | padding...
const headerSize = 32

const (
	offShmSize    = 0
	offConnType   = sizeOfSizeT
	offServerFlag = sizeOfSizeT + 1
	offClientFlag = sizeOfSizeT + 2
)

const offFlagsAligned = sizeOfSizeT

type DMessage struct {
	Size uintptr
	Data []byte
}

type DConnection struct {
	portName        string
	basePtr         unsafe.Pointer
	serverDataPtr   unsafe.Pointer
	clientDataPtr   unsafe.Pointer
	shmSize         uintptr
	connectionType  byte
	identifier      byte // 1 = server (creator), 0 = client
	handle          platformHandle
	lastReadFlagOff uintptr
	hasPendingAck   bool
	mu              sync.Mutex
}

func (c *DConnection) ShmSize() uintptr     { return c.shmSize }
func (c *DConnection) ConnectionType() byte { return c.connectionType }

func Create(portName string, shmSize uintptr) (*DConnection, error) {

	totalSize := uintptr(headerSize) + 2*(shmSize+sizeOfSizeT)

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
	atomicStoreByte(basePtr, offServerFlag, 1)
	atomicStoreByte(basePtr, offClientFlag, 1)

	conn.serverDataPtr = unsafe.Add(basePtr, headerSize)
	conn.clientDataPtr = unsafe.Add(basePtr, headerSize+sizeOfSizeT+shmSize)
	return conn, nil
}

func Connect(portName string) (*DConnection, error) {
	basePtr, handle, err := openShm(portName)
	if err != nil {
		return nil, err
	}

	conn := &DConnection{
		portName:       portName,
		basePtr:        basePtr,
		shmSize:        *(*uintptr)(basePtr),
		connectionType: *(*byte)(unsafe.Add(basePtr, offConnType)),
		identifier:     0,
		handle:         handle,
	}
	conn.serverDataPtr = unsafe.Add(basePtr, headerSize)
	conn.clientDataPtr = unsafe.Add(basePtr, headerSize+sizeOfSizeT+conn.shmSize)
	return conn, nil
}

func (c *DConnection) Close() {
	c.mu.Lock()
	if c.hasPendingAck {
		atomicStoreByte(c.basePtr, c.lastReadFlagOff, stateEmpty)
		c.hasPendingAck = false
	}
	c.mu.Unlock()
	closeShm(c.basePtr, uintptr(headerSize)+2*(c.shmSize+sizeOfSizeT), c.handle, c.identifier == 1)
}

func (c *DConnection) WriteBytes(bytes []byte) error {
	return c.Write(&DMessage{Data: bytes, Size: uintptr(len(bytes))})
}

const (
	stateReady   = 0
	stateEmpty   = 1
	stateReading = 2
	stateWriting = 3
)

func (c *DConnection) Write(msg *DMessage) error {
	if msg.Size > c.shmSize {
		return errors.New("dport: message size exceeds shared memory capacity")
	}

	flagOff := offServerFlag
	writePtr := c.clientDataPtr
	if c.identifier == 1 {
		flagOff = offClientFlag
		writePtr = c.serverDataPtr
	}

	for {
		if atomicCASByte(c.basePtr, flagOff, stateEmpty, stateWriting) {
			break
		}
		runtime.Gosched()
	}

	*(*uintptr)(writePtr) = msg.Size
	copy(
		unsafe.Slice((*byte)(unsafe.Add(writePtr, sizeOfSizeT)), msg.Size),
		msg.Data,
	)

	atomicStoreByte(c.basePtr, flagOff, 0)
	return nil
}

func (c *DConnection) Read() DMessage {
	flagOff := offClientFlag
	readPtr := c.serverDataPtr
	if c.identifier == 1 {
		flagOff = offServerFlag
		readPtr = c.clientDataPtr
	}

	for {
		if atomicCASByte(c.basePtr, flagOff, stateReady, stateReading) {
			break
		}

		c.mu.Lock()
		if c.hasPendingAck {
			atomicStoreByte(c.basePtr, c.lastReadFlagOff, stateEmpty)
			c.hasPendingAck = false
		}
		c.mu.Unlock()

		runtime.Gosched()
	}

	size := *(*uintptr)(readPtr)
	data := make([]byte, size)
	copy(
		data,
		unsafe.Slice((*byte)(unsafe.Add(readPtr, sizeOfSizeT)), size),
	)

	c.mu.Lock()
	c.lastReadFlagOff = flagOff
	c.hasPendingAck = true
	c.mu.Unlock()

	return DMessage{Size: size, Data: data}
}

func atomicCASByte(basePtr unsafe.Pointer, flagOff uintptr, oldVal, newVal byte) bool {
	aligned := (*uint32)(unsafe.Add(basePtr, flagOff&^uintptr(3)))
	shift := uint((flagOff & 3) * 8)
	mask := uint32(0xFF) << shift
	oldBits := uint32(oldVal) << shift
	newBits := uint32(newVal) << shift

	oldUint32 := atomic.LoadUint32(aligned)
	if (oldUint32 & mask) != oldBits {
		return false
	}
	return atomic.CompareAndSwapUint32(aligned, oldUint32, (oldUint32&^mask)|newBits)
}


func atomicStoreByte(basePtr unsafe.Pointer, flagOff uintptr, val byte) {
	aligned := (*uint32)(unsafe.Add(basePtr, flagOff & ^uintptr(3)))
	shift := uint((flagOff & 3) * 8)
	mask := uint32(0xFF) << shift
	bits := uint32(val) << shift

	for {
		old := atomic.LoadUint32(aligned)
		if atomic.CompareAndSwapUint32(aligned, old, (old & ^mask)|bits) {
			return
		}
	}
}
