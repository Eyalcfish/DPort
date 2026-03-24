package dport

import (
	"errors"
	"runtime"
	"sync/atomic"
	"unsafe"
)

const sizeOfSizeT = unsafe.Sizeof(uintptr(0))

// headerSize matches the C packed DConnectionHeader.
// Layout: size_t | char | uchar | uchar | [int futex_flag on Linux]
const headerSize = sizeOfSizeT + 1 + 1 + 1 + headerFutexSize

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
	portName       string
	basePtr        unsafe.Pointer
	serverDataPtr  unsafe.Pointer
	clientDataPtr  unsafe.Pointer
	shmSize        uintptr
	connectionType byte
	identifier     byte // 1 = server (creator), 0 = client
	handle         platformHandle
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
	closeShm(c.basePtr, uintptr(headerSize)+2*(c.shmSize+sizeOfSizeT), c.handle, c.identifier == 1)
}

func (c *DConnection) WriteBytes(bytes []byte) error {
	return c.Write(&DMessage{Data: bytes, Size: uintptr(len(bytes))})
}

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

	spinWaitByte(c.basePtr, flagOff, 1)

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

	spinWaitByte(c.basePtr, flagOff, 0)

	size := *(*uintptr)(readPtr)
	data := make([]byte, size)
	copy(
		data,
		unsafe.Slice((*byte)(unsafe.Add(readPtr, sizeOfSizeT)), size),
	)

	atomicStoreByte(c.basePtr, flagOff, 1)

	return DMessage{Size: size, Data: data}
}

func spinWaitByte(basePtr unsafe.Pointer, flagOff uintptr, target byte) {
	aligned := (*uint32)(unsafe.Add(basePtr, flagOff & ^uintptr(3)))
	shift := uint((flagOff & 3) * 8)
	want := uint32(target) << shift
	mask := uint32(0xFF) << shift

	for i := 0; ; i++ {
		if atomic.LoadUint32(aligned)&mask == want {
			return
		}
		if i&0x3F == 0 { // TODO: choose whatever todo with this
			runtime.Gosched()
		}
	}
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
