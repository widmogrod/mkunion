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

	assert.PanicsWithError(t, `missing params: ":id", extra params: "id"`, func() {
		MustWhere("ID = :id", ParamBinds{"id": schema.MkInt(1)}, nil)
	})

	assert.PanicsWithError(t, `missing params: ":id"`, func() {
		MustWhere("ID = :id", ParamBinds{}, nil)
	})
}

func TestWhereEmptyQueryMatchesAll(t *testing.T) {
	w, err := Where("", nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, w)

	data := schema.MkMap(
		schema.MkField("ID", schema.MkString("123")),
	)
	assert.True(t, w.Evaluate(data))
}

func TestWhereEmptyQueryRejectsExtraParams(t *testing.T) {
	_, err := Where("", ParamBinds{":id": schema.MkInt(1)}, nil)
	assert.ErrorContains(t, err, `extra params: ":id"`)
}

func TestWhereAllowExtraParams(t *testing.T) {
	w, err := Where("ID = :id", ParamBinds{
		":id":    schema.MkInt(1),
		":extra": schema.MkInt(2),
	}, &WhereOpt{AllowExtraParams: true})
	assert.NoError(t, err)
	assert.NotNil(t, w)
}

func TestWhereDuplicateBinds(t *testing.T) {
	assert.NotPanics(t, func() {
		w := MustWhere("ID = :id OR Age = :id", ParamBinds{":id": schema.MkInt(1)}, nil)
		assert.NotNil(t, w)
	})
}
