package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort/src/dport"
)

func main() {
	conn, err := dport.Connect("example_port")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	buf := []byte("asd")
	for i := 0; i < 10; i++ {
		buf[2] = '0' + byte(i%10)
		msg := &dport.DMessage{Size: uintptr(len(buf)), Data: buf}
		fmt.Printf("Sending message: %s\n", buf)
		if err := conn.Write(msg); err != nil {
			panic(err)
		}
	}
}
