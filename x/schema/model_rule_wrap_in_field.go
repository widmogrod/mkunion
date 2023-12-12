package schema

var _ RuleMatcher = (*WrapInMap[any])(nil)

type WrapInMap[A any] struct {
	ForType A
	InField string
}

func (w *WrapInMap[A]) MapDefFor(x *Map, path []string, config *goConfig) (TypeMapDefinition, bool) {
	return nil, false
}

func (w *WrapInMap[A]) SchemaToUnionType(x any, schema Schema, config *goConfig) (Schema, bool) {
	if _, ok := x.(A); !ok {
		return nil, false
	}

	return &Map{
		w.InField: schema,
	}, true
}
