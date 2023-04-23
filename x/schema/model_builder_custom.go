package schema

type SelfUnmarshallingStructBuilder struct {
	new func() Unmarshaler
}

func (c *SelfUnmarshallingStructBuilder) NewMapBuilder() MapBuilder {
	return &selfUnmarshalTypeMapBuilder{
		fs: c.new(),
	}
}

var _ TypeMapDefinition = (*SelfUnmarshallingStructBuilder)(nil)

func UseSelfUnmarshallingStruct(new func() Unmarshaler) *SelfUnmarshallingStructBuilder {
	return &SelfUnmarshallingStructBuilder{
		new: new,
	}
}

var _ MapBuilder = (*selfUnmarshalTypeMapBuilder)(nil)

type selfUnmarshalTypeMapBuilder struct {
	fs Unmarshaler
}

func (c *selfUnmarshalTypeMapBuilder) BuildFromMapSchema(x *Map) (any, error) {
	err := c.fs.UnmarshalSchema(x)
	if err != nil {
		return nil, err
	}

	return c.fs, nil
}

func (c *selfUnmarshalTypeMapBuilder) Set(key string, value any) error {
	panic("schema.selfUnmarshalTypeMapBuilder.Set: should not be called")
}

func (c *selfUnmarshalTypeMapBuilder) Build() any {
	panic("schema.selfUnmarshalTypeMapBuilder.Build: should not be called")
}
