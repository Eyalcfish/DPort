package benchmark

import (
	"fmt"
	"testing"

	dport "github.com/Eyalcfish/DPort"
)

func BenchmarkRoundTrip(b *testing.B) {
	const port = "bench_rt"
	srv, err := dport.Create(port, 1024)
	if err != nil {
		b.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		b.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	payload := make([]byte, 64)
	msg := &dport.DMessage{Size: uintptr(len(payload)), Data: payload}

	done := make(chan struct{})
	go func() {
		for i := 0; i < b.N; i++ {
			srv.Read()
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cli.Write(msg, 1)
	}
	<-done
}

func BenchmarkPingPong(b *testing.B) {
	const port = "bench_pp"
	srv, err := dport.Create(port, 1024)
	if err != nil {
		b.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		b.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	payload := make([]byte, 64)
	msg := &dport.DMessage{Size: uintptr(len(payload)), Data: payload}

	done := make(chan struct{})
	go func() {
		for i := 0; i < b.N; i++ {
			srv.Read()
			srv.Write(msg, 2)
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cli.Write(msg, 1)
		cli.Read()
	}
	<-done
}

func BenchmarkThroughput(b *testing.B) {
	for _, size := range []int{64, 256, 1024, 4096, 100000, 1000000} {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			port := fmt.Sprintf("bench_tp_%d", size)
			srv, err := dport.Create(port, uintptr(size+128)) // Added buffer
			if err != nil {
				b.Fatalf("Create: %v", err)
			}
			defer srv.Close()

			cli, err := dport.Connect(port)
			if err != nil {
				b.Fatalf("Connect: %v", err)
			}
			defer cli.Close()

			payload := make([]byte, size)
			msg := &dport.DMessage{Size: uintptr(size), Data: payload}

			done := make(chan struct{})
			go func() {
				for i := 0; i < b.N; i++ {
					srv.Read()
				}
				close(done)
			}()

			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cli.Write(msg, 1)
			}
			<-done
		})
	}
}

func BenchmarkSimpleTransfer(b *testing.B) {
	const port = "bench_simple"
	payload := []byte("bench-payload!") // 14 bytes

	srv, err := dport.Create(port, 1024)
	if err != nil {
		b.Fatalf("Create: %v", err)
	}
	defer srv.Close()

	cli, err := dport.Connect(port)
	if err != nil {
		b.Fatalf("Connect: %v", err)
	}
	defer cli.Close()

	n := b.N
	msg := &dport.DMessage{Size: uintptr(len(payload)), Data: payload}

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for i := 0; i < n; i++ {
			srv.Read()
		}
	}()

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < n; i++ {
		cli.Write(msg, 1)
	}

	<-readerDone
}
