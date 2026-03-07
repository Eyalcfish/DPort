package main

/*
#include <stdint.h>
#include <stdlib.h>

// DPortMessage is the C-compatible message structure.
typedef struct {
	uint64_t size;
	void*    data;
} DPortMessage;
*/
import "C"
import (
	"sync"
	"unsafe"

	dport "github.com/Eyalcfish/DPort/src/dport"
)

// handleMu protects the handles map.
var handleMu sync.Mutex

// handles maps integer handles to Go DConnection pointers.
// We can't pass Go pointers to C, so we use integer handles instead.
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

// DPort_Read reads a message from a DPort connection.
// The caller must free the returned data pointer with DPort_FreeMessage.
// Returns a DPortMessage with size=0 and data=NULL on failure.
//
//export DPort_Read
func DPort_Read(handle C.int) C.DPortMessage {
	conn := getConn(handle)
	if conn == nil {
		return C.DPortMessage{size: 0, data: nil}
	}

	msg := conn.Read()

	// Allocate C memory and copy the data into it.
	// The caller is responsible for freeing this with DPort_FreeMessage.
	cData := C.malloc(C.size_t(msg.Size))
	copy(
		unsafe.Slice((*byte)(cData), msg.Size),
		msg.Data,
	)

	return C.DPortMessage{
		size: C.uint64_t(msg.Size),
		data: cData,
	}
}

// DPort_FreeMessage frees the data pointer from a DPortMessage returned by DPort_Read.
//
//export DPort_FreeMessage
func DPort_FreeMessage(msg C.DPortMessage) {
	if msg.data != nil {
		C.free(msg.data)
	}
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
