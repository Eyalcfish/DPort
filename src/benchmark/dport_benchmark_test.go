package benchmark

import (
	"fmt"
	"testing"

	dport "github.com/Eyalcfish/DPort/src/dport"
	"github.com/Eyalcfish/DPort/src/dport/queue"
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
		cli.Write(msg)
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
			srv.Write(msg)
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cli.Write(msg)
		cli.Read()
	}
	<-done
}

func BenchmarkThroughput(b *testing.B) {
	for _, size := range []int{64, 256, 1024, 4096} {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			port := fmt.Sprintf("bench_tp_%d", size)
			srv, err := dport.Create(port, uintptr(size+64))
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
				cli.Write(msg)
			}
			<-done
		})
	}
}

func BenchmarkWorkers(b *testing.B) {
	const port = "bench_workers"
	payload := []byte("bench-payload!") // 14 bytes

	srv, err := dport.Create(port, 1024)
	if err != nil {
		b.Fatalf("Create: %v", err)
	}

	cli, err := dport.Connect(port)
	if err != nil {
		srv.Close()
		b.Fatalf("Connect: %v", err)
	}

	head := &queue.Package{}
	n := b.N

	// Reader goroutine: reads exactly n messages into the queue.
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		tail := head
		for i := 0; i < n; i++ {
			msg := srv.Read()
			tail = queue.WriteToPackage(tail, &msg)
		}
	}()

	// Writer goroutine: sends exactly n messages.
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for i := 0; i < n; i++ {
			cli.Write(&dport.DMessage{Size: uintptr(len(payload)), Data: payload})
		}
	}()

	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	// Drain n messages from the queue.
	cur := head
	for i := 0; i < n; i++ {
		var ok bool
		for {
			_, cur, ok = queue.ReadFromPackage(cur)
			if ok {
				break
			}
		}
	}

	b.StopTimer()
	<-readerDone
	<-writerDone
	cli.Close()
	srv.Close()
}
