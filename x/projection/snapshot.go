package projection

import (
	"github.com/widmogrod/mkunion/x/stream"
)

//go:generate go run ../../cmd/mkunion/main.go

//go:tag mkunion:"SnapshotState"
type (
	PullPushContextState struct {
		Offset    *stream.Offset
		Watermark *stream.EventTime
		PullTopic stream.Topic
		PushTopic stream.Topic
	}

	JoinContextState struct {
		Offset1    *stream.Offset
		PullTopic1 stream.Topic

		Offset2    *stream.Offset
		PullTopic2 stream.Topic

		LeftOrRight bool

		PushTopic stream.Topic
		Watermark stream.EventTime
	}
)
