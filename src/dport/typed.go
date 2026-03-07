package dport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
)

// TypedConnection wraps a DConnection and provides typed message support
// using FlatBuffers for serialization. Each message type is identified by
// a uint16 ID and validated against a registered minimum payload size.
type TypedConnection struct {
	conn     *DConnection
	mu       sync.RWMutex
	registry map[uint16]typeInfo
}

type typeInfo struct {
	name    string
	minSize int
}

// TypedHeaderSize is the number of bytes prepended to every typed message (uint16 message ID).
const TypedHeaderSize = 2

// NewTypedConnection wraps an existing DConnection for typed message passing.
func NewTypedConnection(conn *DConnection) *TypedConnection {
	return &TypedConnection{
		conn:     conn,
		registry: make(map[uint16]typeInfo),
	}
}

// Register associates a message ID with a human-readable name and a minimum
// payload size. Both sides (client and server) must register the same IDs.
// The minSize is used for validation when receiving messages.
func (tc *TypedConnection) Register(id uint16, name string, minSize int) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if existing, ok := tc.registry[id]; ok {
		return fmt.Errorf("dport: message ID %d already registered as %q", id, existing.name)
	}

	tc.registry[id] = typeInfo{name: name, minSize: minSize}
	return nil
}

// WriteTyped sends a typed message over the connection. The id identifies
// the message type, and payload contains the FlatBuffers-serialized bytes.
func (tc *TypedConnection) WriteTyped(id uint16, payload []byte) error {
	tc.mu.RLock()
	_, registered := tc.registry[id]
	tc.mu.RUnlock()

	if !registered {
		return fmt.Errorf("dport: message ID %d is not registered", id)
	}

	totalSize := TypedHeaderSize + len(payload)
	buf := make([]byte, totalSize)
	binary.LittleEndian.PutUint16(buf[0:2], id)
	copy(buf[2:], payload)

	return tc.conn.Write(&DMessage{
		Size: uintptr(totalSize),
		Data: buf,
	})
}

// TypedMessage holds a received typed message: its ID and the raw
// FlatBuffers payload bytes (without the 2-byte header).
type TypedMessage struct {
	ID      uint16
	Payload []byte
}

// ReadTyped reads a typed message from the connection, validates the
// message ID is registered and the payload meets the minimum size.
func (tc *TypedConnection) ReadTyped() (TypedMessage, error) {
	raw := tc.conn.Read()

	if raw.Size < uintptr(TypedHeaderSize) {
		return TypedMessage{}, errors.New("dport: typed message too short (missing header)")
	}

	id := binary.LittleEndian.Uint16(raw.Data[0:2])
	payload := raw.Data[2:]

	tc.mu.RLock()
	info, registered := tc.registry[id]
	tc.mu.RUnlock()

	if !registered {
		return TypedMessage{}, fmt.Errorf("dport: received unregistered message ID %d", id)
	}

	if len(payload) < info.minSize {
		return TypedMessage{}, fmt.Errorf(
			"dport: message %q (ID %d): payload size %d < minimum %d",
			info.name, id, len(payload), info.minSize,
		)
	}

	return TypedMessage{ID: id, Payload: payload}, nil
}

// Connection returns the underlying DConnection.
func (tc *TypedConnection) Connection() *DConnection {
	return tc.conn
}

// Close closes the underlying DConnection.
func (tc *TypedConnection) Close() {
	tc.conn.Close()
}
