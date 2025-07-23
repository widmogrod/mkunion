package example

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func TestConnectedOpen_Send(t *testing.T) {
	c := &Connected[Open]{Conn: &mockConn{writeFunc: func(bytes []byte) (int, error) {
		return 0, nil
	}}}

	err := c.Send([]byte(`hello`))
	assert.NoError(t, err)

	c2 := c.Close()
	assert.IsType(t, &Disconnected[Closed]{}, c2)
}

// mockConn is a custom implementation of the net.Conn interface for testing.
type mockConn struct {
	writeFunc func([]byte) (int, error)
}

var _ net.Conn = (*mockConn)(nil)

func (m *mockConn) Write(p []byte) (n int, err error)  { return m.writeFunc(p) }
func (m *mockConn) Read(p []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }
