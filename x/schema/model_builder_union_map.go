package schema

type UnionMap struct {
	last any
}

var (
	_ TypeMapDefinition = (*UnionMap)(nil)
	_ MapBuilder        = (*UnionMap)(nil)
)

func (u *UnionMap) NewMapBuilder() MapBuilder {
	return &UnionMap{}
}

func (u *UnionMap) Set(key string, value any) error {
	u.last = value
	return nil
}

func (u *UnionMap) Build() any {
	return u.last
}
