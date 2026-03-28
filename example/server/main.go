package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort"
)

func main() {
	const portName = "example_port"
	const shmSize = 4096

	fmt.Printf("[Server] Creating port %q (shm size: %d bytes)...\n", portName, shmSize)
	conn, err := dport.Create(portName, shmSize)
	if err != nil {
		fmt.Printf("[Server] ERROR: DPort_Create failed: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("[Server] Port created\n")
	fmt.Println("[Server] Waiting for messages...")

	for {
		// Block until a message arrives
		msg := conn.Read()
		message := string(msg.Data)
		fmt.Printf("[Server] Received: %q\n", message)
	}
}
