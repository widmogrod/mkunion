package schema

var _ ListBuilder = (*NativeList)(nil)

type NativeList struct {
	l []any
}

func (s *NativeList) NewListBuilder() ListBuilder {
	return &NativeList{
		l: nil,
	}
}

func (s *NativeList) Append(value any) error {
	s.l = append(s.l, value)
	return nil
}

func (s *NativeList) Build() any {
	return s.l
}

var _ MapBuilder = (*NativeMap)(nil)

type NativeMap struct {
	m map[string]any
}

func (s *NativeMap) NewMapBuilder() MapBuilder {
	return &NativeMap{
		m: make(map[string]any),
	}
}

func (s *NativeMap) Build() any {
	return s.m
}

func (s *NativeMap) Set(k string, value any) error {
	s.m[k] = value
	return nil
}
