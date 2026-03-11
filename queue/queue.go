package queue

import (
	"sync/atomic"

	dport "github.com/Eyalcfish/DPort"
)

type Package struct {
	Flag uint32
	Next *Package
	Msg  dport.DMessage
}

func StartWorkers(conn *dport.DConnection, killSwitch *uint8) (readQueue *Package, writeQueue *Package) {
	readQueue = &Package{
		Flag: 0,
		Next: nil,
	}

	writeQueue = &Package{
		Flag: 0,
		Next: nil,
	}

	go readWorker(conn, readQueue, killSwitch)
	go writeWorker(conn, writeQueue, killSwitch)

	return readQueue, writeQueue
}

func WriteWithWorker(queue *Package, msg *dport.DMessage) *Package {
	a := &dport.DMessage{
		Data: make([]byte, msg.Size),
		Size: msg.Size,
	}
	copy(a.Data, msg.Data)
	return WriteToPackage(queue, a)
}

func WriteToPackage(queue *Package, msg *dport.DMessage) *Package {
	queue.Msg = *msg
	queue.Next = &Package{
		Flag: 0,
		Next: nil,
	}
	atomic.StoreUint32(&queue.Flag, 1)
	return queue.Next
}

// ReadFromPackage returns the message, the next package, and a boolean indicating if a message was read.
func ReadFromPackage(queue *Package) (dport.DMessage, *Package, bool) {
	if atomic.LoadUint32(&queue.Flag) == 0 {
		return dport.DMessage{}, queue, false
	}
	msg := queue.Msg
	atomic.StoreUint32(&queue.Flag, 0)
	return msg, queue.Next, true
}

func readWorker(conn *dport.DConnection, queue *Package, killSwitch *uint8) {
	for {
		if *killSwitch == 1 {
			break
		}
		msg := conn.Read()
		queue = WriteToPackage(queue, &msg)
	}
}

func writeWorker(conn *dport.DConnection, queue *Package, killSwitch *uint8) {
	for {
		if *killSwitch == 1 {
			break
		}
		msg, nextQueue, ok := ReadFromPackage(queue)
		queue = nextQueue
		if !ok {
			continue
		}
		conn.Write(&msg)
	}
}
