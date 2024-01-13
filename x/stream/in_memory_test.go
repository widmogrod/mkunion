package stream

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewInMemoryStream(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTime)
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
	s := NewInMemoryStream[int](WithSystemTimeFixed(4513))
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
	assert.ErrorAs(t, err, &ErrEmptyCommand)

	value, err = s.Pull(&FromBeginning{})
	assert.NoError(t, err)

	expected := &Item[int]{
		Data:      1,
		EventTime: MkEventTimeFromInt(4513),
		Offset:    MkOffsetFromInt(0),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}

	value, err = s.Pull(value.Offset)
	assert.NoError(t, err)
	assert.Equal(t, 2, value.Data)
	expected = &Item[int]{
		Data:      2,
		EventTime: MkEventTimeFromInt(4513),
		Offset:    MkOffsetFromInt(1),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}

	value, err = s.Pull(value.Offset)
	assert.ErrorAs(t, err, &ErrEndOfStream)
}
