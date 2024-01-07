package projection

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestCountHandler(t *testing.T) {
	h := &CountHandler{}
	assert.Equal(t, 0, h.value)

	l := &ListAssert{t: t}
	err := h.Process(Item{
		Data: schema.MkInt(1),
	}, l.Returning)
	assert.NoError(t, err)
	l.AssertAt(0, Item{
		Data: schema.MkInt(1),
	})
	assert.Equal(t, 1, h.value)

	err = h.Process(Item{
		Data: schema.MkInt(2),
	}, l.Returning)
	assert.NoError(t, err)
	l.AssertAt(1, Item{
		Data: schema.MkInt(3),
	})
	assert.Equal(t, 3, h.value)

}
