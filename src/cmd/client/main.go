package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort/src/dport"
	queue "github.com/Eyalcfish/DPort/src/dport/queue"
)

func main() {
	conn, err := dport.Connect("example_port")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	killSwitch := uint8(0)
	_, writeQueue := queue.StartWorkers(conn, &killSwitch)

	buf := []byte("asd")
	for i := 0; i < 10; i++ {
		buf[2] = '0' + byte(i%10)
		msg := &dport.DMessage{Size: uintptr(len(buf)), Data: buf}
		fmt.Printf("Sending message: %s\n", buf)
		writeQueue = queue.WriteToPackage(writeQueue, msg)
	}
	killSwitch = 1
}
