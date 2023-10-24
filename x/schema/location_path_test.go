package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseLocation(t *testing.T) {
	result, err := ParseLocation(`x.y[1].Data["some.Some"].Abc[*].X`)
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
	}, result)
}

func TestParseLocation2(t *testing.T) {
	result, err := ParseLocation(`Tree[*].Right["testutil.Branch"].Value[*]`)
	assert.NoError(t, err)
	assert.Equal(t, []Location{
		&LocationField{Name: "Tree"},
		&LocationAnything{},
		&LocationField{Name: "Right"},
		&LocationField{Name: "testutil.Branch"},
		&LocationField{Name: "Value"},
		&LocationAnything{},
	}, result)
}
