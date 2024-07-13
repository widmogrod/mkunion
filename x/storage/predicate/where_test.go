package predicate

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestMustWhere(t *testing.T) {
	assert.NotPanics(t, func() {
		MustWhere("ID = :id", ParamBinds{":id": schema.MkInt(1)}, nil)
	})

	assert.Panics(t, func() {
		MustWhere("ID = :id", ParamBinds{"id": schema.MkInt(1)}, nil)
	}, `missing bind params ":id", unknown bind params "id"`)

	assert.Panics(t, func() {
		MustWhere("ID = :id", ParamBinds{}, nil)
	}, `missing bind params ":id"`)
}
