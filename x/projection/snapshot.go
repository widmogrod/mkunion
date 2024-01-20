package projection

import (
	"github.com/widmogrod/mkunion/x/stream"
)

type SnapshotState struct {
	Offset    *stream.Offset
	PullTopic stream.Topic
	PushTopic stream.Topic
}
