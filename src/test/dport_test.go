package test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	dport "github.com/Eyalcfish/DPort/src/dport"
	"github.com/Eyalcfish/DPort/src/dport/queue"
)

func TestWorkers(t *testing.T) {
	const port = "test_workers"
	const n = 10

	srv, err := dport.Create(port, 1024)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cli, err := dport.Connect(port)
	if err != nil {
		srv.Close()
		t.Fatalf("Connect: %v", err)
	}

	// Reader goroutine: reads exactly n messages from the DPort connection
	// and writes them into the queue.
	head := &queue.Package{}
	readerDone := make(chan struct{})

	go func() {
		defer close(readerDone)
		tail := head
		for i := 0; i < n; i++ {
			msg := srv.Read()
			tail = queue.WriteToPackage(tail, &msg)
		}
	}()

	// Writer goroutine: sends exactly n messages through the DPort connection.
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for i := 0; i < n; i++ {
			data := []byte(fmt.Sprintf("msg-%04d", i))
			cli.Write(&dport.DMessage{Size: uintptr(len(data)), Data: data})
		}
	}()

	// Main goroutine: read n messages from the queue.
	cur := head
	for i := 0; i < n; i++ {
		var msg dport.DMessage
		var ok bool
		for {
			msg, cur, ok = queue.ReadFromPackage(cur)
			if ok {
				break
			}
		}
		want := fmt.Sprintf("msg-%04d", i)
		if string(msg.Data) != want {
			t.Fatalf("message %d: got %q, want %q", i, msg.Data, want)
		}
	}

	// Wait for both goroutines to finish before closing connections.
	<-readerDone
	<-writerDone
	cli.Close()
	srv.Close()
}

func TestCreateAndConnect(t *testing.T) {
	const port = "test_create_connect"
	srv, err := dport.Create(port, 256)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	if cli.ShmSize() != 256 {
		t.Errorf("ShmSize = %d, want 256", cli.ShmSize())
	}
	if cli.ConnectionType() != 2 {
		t.Errorf("ConnectionType = %d, want 2", cli.ConnectionType())
	}
}

func TestSingleMessage(t *testing.T) {
	const port = "test_single_msg"
	srv, err := dport.Create(port, 256)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	payload := []byte("hello dport")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cli.Write(&dport.DMessage{Size: uintptr(len(payload)), Data: payload})
	}()

	msg := srv.Read()
	wg.Wait()

	if msg.Size != uintptr(len(payload)) {
		t.Errorf("Size = %d, want %d", msg.Size, len(payload))
	}
	if !bytes.Equal(msg.Data, payload) {
		t.Errorf("Data = %q, want %q", msg.Data, payload)
	}
}

func TestMultipleMessages(t *testing.T) {
	const port = "test_multi_msg"
	const n = 50

	srv, err := dport.Create(port, 1024)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			data := []byte(fmt.Sprintf("msg-%04d", i))
			cli.Write(&dport.DMessage{Size: uintptr(len(data)), Data: data})
		}
	}()

	for i := 0; i < n; i++ {
		msg := srv.Read()
		want := fmt.Sprintf("msg-%04d", i)
		if string(msg.Data) != want {
			t.Fatalf("message %d: got %q, want %q", i, msg.Data, want)
		}
	}
	wg.Wait()
}

func TestBidirectional(t *testing.T) {
	const port = "test_bidir"

	srv, err := dport.Create(port, 256)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cli.Write(&dport.DMessage{Size: 4, Data: []byte("ping")})
		reply := cli.Read()
		if string(reply.Data) != "pong" {
			t.Errorf("client got %q, want %q", reply.Data, "pong")
		}
	}()

	msg := srv.Read()
	if string(msg.Data) != "ping" {
		t.Fatalf("server got %q, want %q", msg.Data, "ping")
	}
	srv.Write(&dport.DMessage{Size: 4, Data: []byte("pong")})

	wg.Wait()
}

func TestWriteTooLarge(t *testing.T) {
	const port = "test_too_large"
	srv, err := dport.Create(port, 16)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	big := make([]byte, 32)
	err = cli.Write(&dport.DMessage{Size: uintptr(len(big)), Data: big})
	if err == nil {
		t.Fatal("expected error for oversized message")
	}
}
