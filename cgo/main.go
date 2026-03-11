package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"
import (
	"sync"
	"unsafe"

	dport "github.com/Eyalcfish/DPort"
)

var handleMu sync.Mutex

var handles = make(map[C.int]*dport.DConnection)
var nextHandle C.int = 1

func storeConn(conn *dport.DConnection) C.int {
	handleMu.Lock()
	defer handleMu.Unlock()
	h := nextHandle
	nextHandle++
	handles[h] = conn
	return h
}

func getConn(h C.int) *dport.DConnection {
	handleMu.Lock()
	defer handleMu.Unlock()
	return handles[h]
}

func removeConn(h C.int) {
	handleMu.Lock()
	defer handleMu.Unlock()
	delete(handles, h)
}

// DPort_Create creates a new DPort server.
// Returns a handle (>0) on success, or 0 on failure.
//
//export DPort_Create
func DPort_Create(portName *C.char, shmSize C.uint64_t) C.int {
	conn, err := dport.Create(C.GoString(portName), uintptr(shmSize))
	if err != nil {
		return 0
	}
	return storeConn(conn)
}

// DPort_Connect connects to an existing DPort server.
// Returns a handle (>0) on success, or 0 on failure.
//
//export DPort_Connect
func DPort_Connect(portName *C.char) C.int {
	conn, err := dport.Connect(C.GoString(portName))
	if err != nil {
		return 0
	}
	return storeConn(conn)
}

// DPort_Close closes a DPort connection and frees the handle.
//
//export DPort_Close
func DPort_Close(handle C.int) {
	conn := getConn(handle)
	if conn != nil {
		conn.Close()
		removeConn(handle)
	}
}

// DPort_Write writes data to a DPort connection.
// Returns 0 on success, -1 on failure.
//
//export DPort_Write
func DPort_Write(handle C.int, data unsafe.Pointer, size C.uint64_t) C.int {
	conn := getConn(handle)
	if conn == nil {
		return -1
	}

	goData := C.GoBytes(data, C.int(size))
	err := conn.Write(&dport.DMessage{
		Size: uintptr(size),
		Data: goData,
	})
	if err != nil {
		return -1
	}
	return 0
}

// DPort_Read reads a message into a caller-provided buffer.
// Returns the message size (bytes written to buf), or -1 on error.
// If the message is larger than bufSize, it is truncated.
//
//export DPort_Read
func DPort_Read(handle C.int, buf unsafe.Pointer, bufSize C.uint64_t) C.longlong {
	conn := getConn(handle)
	if conn == nil {
		return -1
	}

	msg := conn.Read()

	n := msg.Size
	if n > uintptr(bufSize) {
		n = uintptr(bufSize)
	}

	copy(
		unsafe.Slice((*byte)(buf), n),
		msg.Data[:n],
	)

	return C.longlong(msg.Size)
}

// DPort_ShmSize returns the shared memory size for a connection.
//
//export DPort_ShmSize
func DPort_ShmSize(handle C.int) C.uint64_t {
	conn := getConn(handle)
	if conn == nil {
		return 0
	}
	return C.uint64_t(conn.ShmSize())
}

// DPort_ConnectionType returns the connection type byte.
//
//export DPort_ConnectionType
func DPort_ConnectionType(handle C.int) C.char {
	conn := getConn(handle)
	if conn == nil {
		return 0
	}
	return C.char(conn.ConnectionType())
}

func main() {}
