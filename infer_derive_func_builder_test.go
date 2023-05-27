package mkunion

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatchBuilding(t *testing.T) {
	builder := NewMatchBuilder()

	err := builder.SetName("MatchAlphabetNumber")
	assert.NoError(t, err)

	err = builder.SetInputs("Alphabet", "Number")
	assert.NoError(t, err)

	err = builder.AddCase("Name1", "A", "N0")
	assert.NoError(t, err)

	err = builder.AddCase("Name2", "C", "any")
	assert.NoError(t, err)

	err = builder.AddCase("Name3", "any", "any")
	assert.NoError(t, err)

	output, err := builder.Build()
	assert.NoError(t, err)

	assert.Equal(t, [][]string{
		{"A", "N0"},
		{"C", "any"},
		{"any", "any"},
	}, output.Cases)
	assert.Equal(t, []string{"Name1", "Name2", "Name3"}, output.Names)
	assert.Equal(t, []string{"Alphabet", "Number"}, output.Inputs)
	assert.Equal(t, "MatchAlphabetNumber", output.Name)

}
