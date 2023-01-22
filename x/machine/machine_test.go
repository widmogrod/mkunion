package machine

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMachine(t *testing.T) {
	m := NewSimpleMachineWithState(func(cmd string, state int) (int, error) {
		switch cmd {
		case "inc":
			return state + 1, nil
		case "dec":
			return state - 1, nil
		default:
			return 0, fmt.Errorf("unknown cmd: %s", cmd)
		}
	}, 10)

	assert.Equal(t, 10, m.State())

	err := m.Handle("inc")
	assert.NoError(t, err)
	assert.Equal(t, 11, m.State())

	err = m.Handle("dec")
	assert.NoError(t, err)
	assert.Equal(t, 10, m.State())

	err = m.Handle("unknown")
	assert.Error(t, err)
	assert.Equal(t, 10, m.State())
}
