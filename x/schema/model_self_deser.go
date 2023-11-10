package schema

func (x *Map) UnmarshalSchema(y *Map) error {
	// this one feature allow to recursively unmarshal schema
	*x = *y
	return nil
}

func (self *Bool) UnmarshalSchema(x *Map) error {
	*self = *x.Field[0].Value.(*Bool)

	return nil
}

func (self *Binary) UnmarshalSchema(x *Map) error {
	*self = *x.Field[0].Value.(*Binary)

	return nil
}

func (self *Number) UnmarshalSchema(x *Map) error {
	*self = *x.Field[0].Value.(*Number)

	return nil
}

func (self *String) UnmarshalSchema(x *Map) error {
	*self = *x.Field[0].Value.(*String)

	return nil
}

func SchemaSchemaDef() *UnionVariants[Schema] {
	return MustDefineUnion[Schema](
		&None{},
		(*Bool)(nil),
		(*Number)(nil),
		(*String)(nil),
		&Binary{},
		&List{},
		&Map{},
	)
}
