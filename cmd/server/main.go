package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort/dport"
)

func main() {
	conn, err := dport.Create("example_port", 1024)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for i := 0; i < 10; i++ {
		msg := conn.Read()
		fmt.Printf("Size: %d | Data: %s\n", msg.Size, msg.Data)
	}
}
