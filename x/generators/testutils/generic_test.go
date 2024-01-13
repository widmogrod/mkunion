package testutils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGeneric(t *testing.T) {
	var _ Record[int] = &Item[int]{}
	var _ Record[float64] = &Item[float64]{}

	x := &Item[int]{
		Key:  "foo",
		Data: 42,
	}

	y := MatchRecordR1[int](
		x,
		func(x *Item[int]) string {
			return fmt.Sprintf("%s: %d", x.Key, x.Data)
		},
		func(x *Other[int]) string {
			return fmt.Sprintf("%d", x.ValueOf)
		},
	)

	assert.Equal(t, "foo: 42", y)
}
