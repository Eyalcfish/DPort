package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort/src/dport"
	"github.com/Eyalcfish/DPort/src/dport/queue"
)

func main() {
	conn, err := dport.Create("example_port", 1024)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	killSwitch := uint8(0)
	readQueue, _ := queue.StartWorkers(conn, &killSwitch)

	var msg dport.DMessage
	var ok bool
	for i := 0; i < 10; i++ {
		msg, readQueue, ok = queue.ReadFromPackage(readQueue)
		if !ok {
			i--
			continue
		}
		fmt.Printf("Size: %d | Data: %s\n", msg.Size, msg.Data)
	}
}
