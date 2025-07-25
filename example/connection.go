package example

import "net"

// --8<-- [start:example]

//go:tag mkunion:"Connection[State]"
type (
	Disconnected[State any] struct{}
	Connecting[State any]   struct{ Addr string }
	Connected[State any]    struct{ Conn net.Conn }
)

// Type-safe state machine with phantom types
type Unopened struct{}
type Open struct{}
type Closed struct{}

// Only allow certain operations in specific states
func (c *Connected[Open]) Send(data []byte) error {
	// Can only send on open connections
	_, err := c.Conn.Write(data)
	return err
}

func (c *Connected[Open]) Close() Connection[Closed] {
	c.Conn.Close()
	return &Disconnected[Closed]{}
}

// Compile error: cannot call Send on closed connection!
// func (c *Connected[Closed]) Send(data []byte) error { ... }

// --8<-- [end:example]
