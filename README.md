# DPort

A dead-simple, ultra-low latency shared memory IPC library. Written in C, ported to pure Go (no cgo needed).

I built this because I needed a way to blast data between processes as fast as possible without the overhead of sockets or named pipes. It uses pure spin locks on shared memory flags for synchronization perfect for tightly coupled services where microsecond latency matters and you don't mind burning a thread.

### Features
* **Zero CGO:** The Go port uses `x/sys` raw syscalls. 
* **Cross-platform:** Works on Windows and Linux (`/dev/shm` + `mmap`).
* **Binary Compatible:** You can swap data directly between the C and Go implementations. The memory layout is strictly identical.

### Usage

**Server (Go)**
```go
package main

import (
	"fmt"
	"github.com/Eyalcfish/DPort"
)

func main() {
	// Create a 1024-byte shared memory region
	conn, err := dport.Create("my_port", 1024)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Busy-wait for a message
	msg := conn.Read()
	fmt.Printf("Got %d bytes: %s\n", msg.Size, string(msg.Data))
}
```

**Client (Go)**
```go
package main

import (
	"github.com/Eyalcfish/DPort"
)

func main() {
	conn, err := dport.Connect("my_port")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Spin until the server is ready, then write
	conn.Write(&dport.DMessage{
		Size: 5,
		Data: []byte("hello"),
	})
}
```

### Testing

```bash
# Windows
test.bat

# Linux
bash test.bash
```

### Note on CPU Usage
Because this uses busy waiting (spinlock), the reading/writing goroutines will pin a CPU core to 100%. The Go port strategically uses `runtime.Gosched()` to prevent the Go scheduler from starving other goroutines, but it's still fundamentally a hot loop. Use this for high-frequency trading/gaming/realtime stuff, not for web servers.
