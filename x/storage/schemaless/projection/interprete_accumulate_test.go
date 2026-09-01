package projection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

// mapAccumulate merges each windowed item with the previously stored
// aggregate; earlier tests covered only Discard and
// AccumulatingAndRetracting. This drives the method directly, so no
// DAG timing is involved.
func TestMapAccumulateMergesWithPreviousAggregate(t *testing.T) {
	interp := NewInMemoryTwoInterpreter()

	var seen []schema.Schema
	node := &DoMap{
		Ctx: NewContextBuilder(),
		OnMap: &SimpleProcessHandler{
			P: func(x Item, returning func(Item)) error {
				seen = append(seen, x.Data)
				returning(Item{
					Key:       x.Key,
					Data:      x.Data,
					EventTime: x.EventTime,
					Window:    x.Window,
				})
				return nil
			},
		},
	}

	require.NoError(t, interp.pubsub.Register(node))
	go func() {
		// drain published messages so Publish never blocks
		_ = interp.pubsub.Subscribe(context.Background(), node, 0, func(Message) error { return nil })
	}()

	window := &Window{Start: 0, End: 100}
	item := func(value int64) Item {
		return Item{
			Key:       "k",
			Data:      schema.MkInt(value),
			EventTime: 10,
			Window:    window,
		}
	}
	const key = "test-namespace:k"

	t.Run("first item carries only Current and is stored", func(t *testing.T) {
		require.NoError(t, interp.mapAccumulate(context.Background(), node, item(1), key))

		require.Len(t, seen, 1)
		_, hasCurrent := schema.GetSchema(seen[0], "Current")
		_, hasPrevious := schema.GetSchema(seen[0], "Previous")
		assert.True(t, hasCurrent)
		assert.False(t, hasPrevious, "no aggregate exists yet")

		stored, err := interp.bagItem.Get(key)
		require.NoError(t, err)
		assert.Equal(t, "k", stored.Key)
	})

	t.Run("second item carries Previous and Current", func(t *testing.T) {
		require.NoError(t, interp.mapAccumulate(context.Background(), node, item(2), key))

		require.Len(t, seen, 2)
		_, hasCurrent := schema.GetSchema(seen[1], "Current")
		_, hasPrevious := schema.GetSchema(seen[1], "Previous")
		assert.True(t, hasCurrent)
		assert.True(t, hasPrevious, "the stored aggregate must be offered back")

		current, _ := schema.GetSchema(seen[1], "Current")
		assert.Equal(t, schema.MkInt(2), current)
	})
}
