package stream

import (
	"fmt"
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

	value, err = s.Pull(&FromOffset{
		Offset: MkOffsetFromInt(1),
	})
	assert.ErrorAs(t, err, &ErrEndOfStream)
	assert.Nil(t, value)

	item, err := s.Pull(&FromBeginning{
		Topic: "not-exists",
	})
	assert.ErrorAs(t, err, &ErrNoTopicWithName)
	assert.Nil(t, item)

	item, err = s.Pull(&FromOffset{
		Topic:  "not-exists",
		Offset: MkOffsetFromInt(0),
	})
	assert.ErrorAs(t, err, &ErrNoTopicWithName)
	assert.Nil(t, item)
}

func TestInMemoryStream_Push(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTime)
	err := s.Push(&Item[int]{
		Topic:  "topic-1",
		Key:    "asdf",
		Data:   123,
		Offset: MkOffsetFromInt(33),
	})
	assert.ErrorAs(t, err, &ErrOffsetSetOnPush)
}

func TestInMemoryStream_HappyPath(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTimeFixed(4513))
	if s == nil {
		t.Fatalf("NewInMemoryStream: got nil")
	}

	err := s.Push(&Item[int]{
		Topic: "topic-1",
		Key:   "key-1",
		Data:  1,
	})
	assert.NoError(t, err)

	err = s.Push(&Item[int]{
		Topic: "topic-1",
		Key:   "key-2",
		Data:  2,
	})
	assert.NoError(t, err)

	err = s.Push(&Item[int]{
		Data: 3,
	})
	assert.ErrorAs(t, err, &ErrEmptyTopic)

	err = s.Push(&Item[int]{
		Topic:  "topic-1",
		Data:   3,
		Offset: MkOffsetFromInt(123),
	})
	assert.ErrorAs(t, err, &ErrOffsetSetOnPush)

	value, err := s.Pull(nil)
	assert.ErrorAs(t, err, &ErrEmptyCommand)

	_, err = s.Pull(&FromBeginning{})
	assert.ErrorAs(t, err, &ErrEmptyTopic)

	value, err = s.Pull(&FromBeginning{
		Topic: "topic-1",
	})
	assert.NoError(t, err)

	expected := &Item[int]{
		Topic:     "topic-1",
		Key:       "key-1",
		Data:      1,
		EventTime: MkEventTimeFromInt(4513),
		Offset:    MkOffsetFromInt(0),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}

	_, err = s.Pull(&FromOffset{
		Offset: value.Offset,
	})
	assert.ErrorAs(t, err, &ErrEndOfStream)

	value, err = s.Pull(&FromOffset{
		Topic:  "topic-1",
		Offset: value.Offset,
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, value.Data)
	expected = &Item[int]{
		Topic:     "topic-1",
		Key:       "key-2",
		Data:      2,
		EventTime: MkEventTimeFromInt(4513),
		Offset:    MkOffsetFromInt(1),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}

	value, err = s.Pull(&FromOffset{
		Offset: value.Offset,
	})
	assert.ErrorAs(t, err, &ErrEndOfStream)
}

func TestInMemoryStream_TestPublishingOnTwoTopics(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTimeFixed(4513))
	if s == nil {
		t.Fatalf("NewInMemoryStream: got nil")
	}

	err := s.Push(&Item[int]{
		Topic: "topic-1",
		Key:   "key-1",
		Data:  1,
	})
	assert.NoError(t, err)

	err = s.Push(&Item[int]{
		Topic: "topic-2",
		Key:   "key-2",
		Data:  2,
	})
	assert.NoError(t, err)

	value, err := s.Pull(&FromBeginning{
		Topic: "topic-1",
	})
	assert.NoError(t, err)

	expected := &Item[int]{
		Topic:     "topic-1",
		Key:       "key-1",
		Data:      1,
		EventTime: MkEventTimeFromInt(4513),
		Offset:    MkOffsetFromInt(0),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}

	value, err = s.Pull(&FromBeginning{
		Topic: "topic-2",
	})
	assert.NoError(t, err)

	expected = &Item[int]{
		Topic:     "topic-2",
		Key:       "key-2",
		Data:      2,
		EventTime: MkEventTimeFromInt(4513),
		Offset:    MkOffsetFromInt(0),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}
}

func TestInMemoryStream_SimulateRuntimeProblem(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTime)
	if s == nil {
		t.Fatalf("NewInMemoryStream: got nil")
	}

	var customerError1 = fmt.Errorf("customer error 1")
	var customerError2 = fmt.Errorf("customer error 2")

	s.SimulateRuntimeProblem(&SimulateProblem{
		ErrorOnPushProbability: 1,
		ErrorOnPush:            customerError1,
		ErrorOnPull:            customerError2,
	})

	err := s.Push(&Item[int]{
		Topic: "topic-1",
		Key:   "123",
		Data:  123,
	})
	assert.ErrorAs(t, err, &ErrSimulatedError)
	assert.ErrorAs(t, err, &customerError1)

	_, err = s.Pull(&FromBeginning{
		Topic: "topic-1",
	})

	assert.ErrorAs(t, err, &ErrSimulatedError)
	assert.ErrorAs(t, err, &customerError2)
}
