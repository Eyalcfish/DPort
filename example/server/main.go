package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort"
)

func main() {
	const portName = "dart_demo"
	const shmSize = 4096

	fmt.Printf("[Server] Creating port %q (shm size: %d bytes)...\n", portName, shmSize)
	conn, err := dport.Create(portName, shmSize)
	if err != nil {
		fmt.Printf("[Server] ERROR: DPort_Create failed: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("[Server] Port created\n")
	fmt.Println("[Server] Waiting for client to send a message...")

	// Block until a message arrives
	msg := conn.Read()
	message := string(msg.Data)
	fmt.Printf("[Server] Received: %q\n", message)

	// Send a reply
	reply := fmt.Sprintf("Hello from server! I got your message: %q", message)
	fmt.Println("[Server] Sending reply...")
	replyMsg := &dport.DMessage{Size: uintptr(len(reply)), Data: []byte(reply)}
	if err := conn.Write(replyMsg); err != nil {
		fmt.Printf("[Server] ERROR: DPort_Write failed: %v\n", err)
	} else {
		fmt.Println("[Server] Reply sent successfully.")
	}

	// Keep alive briefly so the client can read the reply
	// time.Sleep(2 * time.Second)

	fmt.Println("[Server] Done.")
}
