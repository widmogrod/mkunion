package projection

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

// The accumulate flush mode merges every new windowed value with the
// previous aggregate; earlier tests covered only Discard and
// AccumulatingAndRetracting.
func TestInMemoryInterpreterAccumulate(t *testing.T) {
	dag := NewDAGBuilder()
	loaded := dag.
		Load(&GenerateHandler{
			Load: func(push func(message Item)) error {
				for item := range GenerateItemsEvery(withTime(10, 0), 6, 5*time.Millisecond) {
					push(item)
				}
				return nil
			},
		})

	windowed := loaded.
		Window(
			WithFixedWindow(20*time.Millisecond),
			WithTriggers(&AtWatermark{}),
		)

	var mu sync.Mutex
	var processed int
	windowed.
		Map(&SimpleProcessHandler{
			P: func(x Item, returning func(Item)) error {
				mu.Lock()
				processed++
				mu.Unlock()

				// under accumulation the handler always receives at least
				// the Current field; Previous appears on re-flushes
				_, hasCurrent := schema.GetSchema(x.Data, "Current")
				assert.True(t, hasCurrent, "accumulated items carry a Current field")

				returning(Item{
					Key:       x.Key,
					Data:      x.Data,
					EventTime: x.EventTime,
					Window:    x.Window,
				})
				return nil
			},
		}, WithAccumulate())

	interpret := NewInMemoryTwoInterpreter()
	err := interpret.Run(context.Background(), dag.Build())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, processed, 0, "windows must reach the accumulating map")
}
