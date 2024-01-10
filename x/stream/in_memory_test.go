package stream

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewInMemoryStream(t *testing.T) {
	s := NewInMemoryStream[int]()
	if s == nil {
		t.Fatalf("NewInMemoryStream: got nil")
	}

	value, err := s.Pull(&FromBeginning{})
	assert.ErrorAs(t, err, &ErrEndOfStream)
	assert.Nil(t, value)

	value, err = s.Pull(MkOffsetFromInt(1))
	assert.ErrorAs(t, err, &ErrEndOfStream)
	assert.Nil(t, value)
}

func TestInMemoryStream_Push(t *testing.T) {
	s := NewInMemoryStream[int]()
	if s == nil {
		t.Fatalf("NewInMemoryStream: got nil")
	}

	err := s.Push(&Item[int]{Data: 1})
	assert.NoError(t, err)

	err = s.Push(&Item[int]{Data: 2})
	assert.NoError(t, err)

	err = s.Push(&Item[int]{Data: 3, Offset: MkOffsetFromInt(123)})
	assert.ErrorAs(t, err, &ErrOffsetSetOnPush)

	value, err := s.Pull(nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, value.Data)

	value, err = s.Pull(&FromBeginning{})
	assert.NoError(t, err)
	assert.Equal(t, 1, value.Data)

	value, err = s.Pull(value.Offset)
	assert.NoError(t, err)
	assert.Equal(t, 2, value.Data)

	value, err = s.Pull(value.Offset)
	assert.ErrorAs(t, err, &ErrEndOfStream)
}
