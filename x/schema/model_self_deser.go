package schema

func (x *Map) UnmarshalSchema(y *Map) error {
	// this one feature allow to recursively unmarshal schema
	*x = *y
	return nil
}

func (self *Bool) UnmarshalSchema(x *Map) error {
	for _, value := range *x {
		*self = *value.(*Bool)
		return nil
	}

	return nil
}

func (self *Binary) UnmarshalSchema(x *Map) error {
	for _, value := range *x {
		*self = *value.(*Binary)
		return nil
	}

	return nil
}

func (self *Number) UnmarshalSchema(x *Map) error {
	for _, value := range *x {
		*self = *value.(*Number)
		return nil
	}

	return nil
}

func (self *String) UnmarshalSchema(x *Map) error {
	for _, value := range *x {
		*self = *value.(*String)
		return nil
	}

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
