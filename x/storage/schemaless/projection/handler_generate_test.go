package projection

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestGenerateHandler(t *testing.T) {
	generate := []Item{
		{
			Data: schema.FromGo(Game{
				Players: []string{"a", "b"},
				Winner:  "a",
			}),
		},
		{
			Data: schema.FromGo(Game{
				Players: []string{"a", "b"},
				Winner:  "b",
			}),
		},
		{
			Data: schema.FromGo(Game{
				Players: []string{"a", "b"},
				IsDraw:  true,
			}),
		},
	}

	h := &GenerateHandler{
		Load: func(returning func(message Item)) error {
			for _, msg := range generate {
				returning(msg)
			}
			return nil
		},
	}

	l := &ListAssert{
		t: t,
	}
	err := h.Process(Item{}, l.Returning)
	assert.NoError(t, err)

	l.AssertLen(3)

	for idx, msg := range generate {
		l.AssertAt(idx, msg)
	}
}
