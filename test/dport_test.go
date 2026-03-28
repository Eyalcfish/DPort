package test

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	dport "github.com/Eyalcfish/DPort"
)

func TestWorkers(t *testing.T) {
	const port = "test_workers"
	const n = 1000

	srv, err := dport.Create(port, 1024)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cli, err := dport.Connect(port)
	if err != nil {
		srv.Close()
		t.Fatalf("Connect: %v", err)
	}

	received := make([]dport.DMessage, n)
	readerDone := make(chan struct{})

	go func() {
		defer close(readerDone)
		for i := 0; i < n; i++ {
			received[i] = srv.Read()
		}
	}()

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for i := 0; i < n; i++ {
			data := []byte(fmt.Sprintf("msg-%04d", i))
			cli.Write(&dport.DMessage{Size: uintptr(len(data)), Data: data}, 1)
		}
	}()

	<-readerDone
	<-writerDone

	for i := 0; i < n; i++ {
		want := fmt.Sprintf("msg-%04d", i)
		if string(received[i].Data) != want {
			t.Fatalf("message %d: got %q, want %q", i, received[i].Data, want)
		}
	}

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
		cli.Write(&dport.DMessage{Size: uintptr(len(payload)), Data: payload}, 1)
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
			cli.Write(&dport.DMessage{Size: uintptr(len(data)), Data: data}, 1)
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
		cli.Write(&dport.DMessage{Size: 4, Data: []byte("ping")}, 1)
		reply := cli.Read()
		if string(reply.Data) != "pong" {
			t.Errorf("client got %q, want %q", reply.Data, "pong")
		}
	}()

	msg := srv.Read()
	if string(msg.Data) != "ping" {
		t.Fatalf("server got %q, want %q", msg.Data, "ping")
	}
	srv.Write(&dport.DMessage{Size: 4, Data: []byte("pong")}, 2)

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
	err = cli.Write(&dport.DMessage{Size: uintptr(len(big)), Data: big}, 1)
	if err == nil {
		t.Fatal("expected error for oversized message")
	}
}
