package projection

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestMapHandler(t *testing.T) {
	h := MapGameToStats()
	l := &ListAssert{
		t: t,
	}

	err := h.Process(Item{
		Key: "game:1",
		Data: schema.FromGo(Game{
			Players: []string{"a", "b"},
			Winner:  "a",
		}),
	}, l.Returning)
	assert.NoError(t, err)
	l.AssertLen(2)
	l.AssertAt(0, Item{
		Key: "session-stats-by-player:a",
		Data: schema.FromGo(SessionsStats{
			Wins:  1,
			Loose: 0,
			Draws: 0,
		}),
	})
	l.AssertAt(1, Item{
		Key: "session-stats-by-player:b",
		Data: schema.FromGo(SessionsStats{
			Wins:  0,
			Loose: 1,
			Draws: 0,
		}),
	})
}
