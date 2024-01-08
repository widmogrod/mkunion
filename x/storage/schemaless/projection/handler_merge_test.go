package projection

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestMergeHandler(t *testing.T) {
	h := MergeSessionStats()

	l := &ListAssert{t: t}
	err := h.Process(Item{
		Key: "session-stats-by-player:a",
		Data: schema.MkList(
			schema.FromGo(SessionsStats{
				Wins:  1,
				Draws: 2,
			}),
			schema.FromGo(SessionsStats{
				Wins:  3,
				Draws: 4,
			}),
		),
	}, l.Returning)

	assert.NoError(t, err)
	l.AssertAt(0, Item{
		Key: "session-stats-by-player:a",
		Data: schema.FromGo(SessionsStats{
			Wins:  4,
			Draws: 6,
		}),
	})
}
