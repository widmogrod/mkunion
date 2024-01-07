package projection

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestAvgHandler(t *testing.T) {
	h := &AvgHandler{}
	assert.Equal(t, float64(0), h.avg)

	l := ListAssert{t: t}

	err := h.Process(Item{
		Data: schema.MkInt(1),
	}, l.Returning)
	assert.NoError(t, err)
	l.AssertAt(0, Item{
		Data: schema.MkFloat(1),
	})
	assert.Equal(t, float64(1), h.avg)
	assert.Equal(t, 1, h.count)

	err = h.Process(Item{
		Data: schema.MkInt(11),
	}, l.Returning)
	assert.NoError(t, err)
	l.AssertAt(1, Item{
		Data: schema.MkFloat(6),
	})
	assert.Equal(t, float64(6), h.avg)
	assert.Equal(t, 2, h.count)

	err = h.Process(Item{
		Data: schema.MkInt(3),
	}, l.Returning)
	assert.NoError(t, err)
	l.AssertAt(2, Item{
		Data: schema.MkFloat(5),
	})
	assert.Equal(t, float64(5), h.avg)
	assert.Equal(t, 3, h.count)

}
