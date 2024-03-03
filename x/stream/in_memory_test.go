package stream

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestNewInMemoryStream(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTime)
	if s == nil {
		t.Fatalf("NewInMemoryStream: got nil")
	}

	value, err := s.Pull(&FromBeginning{})
	assert.ErrorIs(t, err, ErrEmptyTopic)
	assert.Nil(t, value)

	value, err = s.Pull(&FromOffset{
		Offset: mkInMemoryOffsetFromInt(1),
	})
	assert.ErrorIs(t, err, ErrEmptyTopic)
	assert.Nil(t, value)

	item, err := s.Pull(&FromBeginning{
		Topic: "not-exists",
	})
	assert.ErrorIs(t, err, ErrNoTopicWithName)
	assert.Nil(t, item)

	item, err = s.Pull(&FromOffset{
		Topic:  "not-exists",
		Offset: mkInMemoryOffsetFromInt(0),
	})
	assert.ErrorIs(t, err, ErrNoTopicWithName)
	assert.Nil(t, item)
}

func TestInMemoryStream_Push(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTime)
	err := s.Push(&Item[int]{
		Topic:  "topic-1",
		Key:    "asdf",
		Data:   123,
		Offset: mkInMemoryOffsetFromInt(33),
	})
	assert.ErrorIs(t, err, ErrOffsetSetOnPush)
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
	assert.ErrorIs(t, err, ErrEmptyTopic)

	err = s.Push(&Item[int]{
		Topic:  "topic-1",
		Data:   3,
		Offset: mkInMemoryOffsetFromInt(123),
	})
	assert.ErrorIs(t, err, ErrEmptyKey)

	value, err := s.Pull(nil)
	assert.ErrorIs(t, err, ErrEmptyCommand)

	_, err = s.Pull(&FromBeginning{})
	assert.ErrorIs(t, err, ErrEmptyTopic)

	value, err = s.Pull(&FromBeginning{
		Topic: "topic-1",
	})
	assert.NoError(t, err)

	expected := &Item[int]{
		Topic:     "topic-1",
		Key:       "key-1",
		Data:      1,
		EventTime: MkEventTimeFromInt(4513),
		Offset:    mkInMemoryOffsetFromInt(0),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}

	_, err = s.Pull(&FromOffset{
		Offset: value.Offset,
	})
	assert.ErrorIs(t, err, ErrEmptyTopic)

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
		Offset:    mkInMemoryOffsetFromInt(1),
	}
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
	}

	value, err = s.Pull(&FromOffset{
		Offset: value.Offset,
	})
	assert.ErrorIs(t, err, ErrEmptyTopic)
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
		Offset:    mkInMemoryOffsetFromInt(0),
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
		Offset:    mkInMemoryOffsetFromInt(0),
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
		ErrorOnPush:            customerError1,
		ErrorOnPushProbability: 1,
		ErrorOnPull:            customerError2,
		ErrorOnPullProbability: 1,
	})

	err := s.Push(&Item[int]{
		Topic: "topic-1",
		Key:   "123",
		Data:  123,
	})
	assert.ErrorIs(t, err, ErrSimulatedError)
	assert.ErrorIs(t, err, customerError1)

	_, err = s.Pull(&FromBeginning{
		Topic: "topic-1",
	})

	t.Log(err.Error())
	assert.ErrorIs(t, err, ErrSimulatedError)
	assert.ErrorIs(t, err, customerError2)
}

func TestStreamHappyPathSpec(t *testing.T) {
	s := NewInMemoryStream[int](WithSystemTimeFixed(10))
	HappyPathSpec(t, s, func() int {
		return rand.Int()
	})
}
