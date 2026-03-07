package test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	dport "github.com/Eyalcfish/DPort/src/dport"
)

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
