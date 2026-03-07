package main

import (
	"fmt"

	dport "github.com/Eyalcfish/DPort/src/dport"

	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/Eyalcfish/DPort/examples/schemas/example"
)

const MsgPlayerPosition uint16 = 1

func main() {
	conn, err := dport.Connect("typed_example")
	if err != nil {
		panic(err)
	}

	tc := dport.NewTypedConnection(conn)
	defer tc.Close()

	// Register the same message types as the server
	tc.Register(MsgPlayerPosition, "PlayerPosition", 12) // 3 × float32 = 12 bytes minimum

	// Build a FlatBuffer with a PlayerPosition
	builder := flatbuffers.NewBuilder(64)

	positions := [][3]float32{
		{1.0, 2.0, 3.0},
		{4.5, 5.5, 6.5},
		{7.0, 8.0, 9.0},
	}

	for _, pos := range positions {
		builder.Reset()

		example.PlayerPositionStart(builder)
		example.PlayerPositionAddX(builder, pos[0])
		example.PlayerPositionAddY(builder, pos[1])
		example.PlayerPositionAddZ(builder, pos[2])
		off := example.PlayerPositionEnd(builder)
		builder.Finish(off)

		payload := builder.FinishedBytes()
		fmt.Printf("Sending PlayerPosition(%.1f, %.1f, %.1f)\n", pos[0], pos[1], pos[2])

		err := tc.WriteTyped(MsgPlayerPosition, payload)
		if err != nil {
			panic(err)
		}
	}
}
