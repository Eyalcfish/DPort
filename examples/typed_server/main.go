package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort/src/dport"

	"github.com/Eyalcfish/DPort/examples/schemas/example"
)

const MsgPlayerPosition uint16 = 1

func main() {
	conn, err := dport.Create("typed_example", 1024)
	if err != nil {
		panic(err)
	}

	tc := dport.NewTypedConnection(conn)
	defer tc.Close()

	// Register message types — must match the client
	tc.Register(MsgPlayerPosition, "PlayerPosition", 12) // 3 × float32 = 12 bytes minimum

	fmt.Println("Typed server listening...")

	for i := 0; i < 3; i++ {
		msg, err := tc.ReadTyped()
		if err != nil {
			panic(err)
		}

		switch msg.ID {
		case MsgPlayerPosition:
			pos := example.GetRootAsPlayerPosition(msg.Payload, 0)
			fmt.Printf("Received PlayerPosition: (%.1f, %.1f, %.1f)\n", pos.X(), pos.Y(), pos.Z())
		default:
			fmt.Printf("Unknown message ID: %d\n", msg.ID)
		}
	}
}
