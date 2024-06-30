package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseLocation(t *testing.T) {
	input := `x.y[1].Data["some.Some"].Abc[*].X[0]["some.Other"]`
	result, err := ParseLocation(input)
	assert.NoError(t, err)
	assert.Equal(t, []Location{
		&LocationField{Name: "x"},
		&LocationField{Name: "y"},
		&LocationIndex{Index: 1},
		&LocationField{Name: "Data"},
		&LocationField{Name: "some.Some"},
		&LocationField{Name: "Abc"},
		&LocationAnything{},
		&LocationField{Name: "X"},
		&LocationIndex{Index: 0},
		&LocationField{Name: "some.Other"},
	}, result)

	assert.Equal(t, input, LocationToStr(result))
}

func TestParseLocation2(t *testing.T) {
	input := `Tree[*].Right["testutil.Branch"].Value[*]`
	result, err := ParseLocation(input)
	assert.NoError(t, err)
	assert.Equal(t, []Location{
		&LocationField{Name: "Tree"},
		&LocationAnything{},
		&LocationField{Name: "Right"},
		&LocationField{Name: "testutil.Branch"},
		&LocationField{Name: "Value"},
		&LocationAnything{},
	}, result)
	assert.Equal(t, input, LocationToStr(result))
}

func TestParseLocationSpecialCases(t *testing.T) {
	input := `Tree["$type"].Value[*]`
	result, err := ParseLocation(input)
	assert.NoError(t, err)
	assert.Equal(t, []Location{
		&LocationField{Name: "Tree"},
		&LocationField{Name: "$type"},
		&LocationField{Name: "Value"},
		&LocationAnything{},
	}, result)
	assert.Equal(t, input, LocationToStr(result))
}
