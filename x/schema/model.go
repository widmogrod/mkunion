package schema

import "github.com/widmogrod/mkunion/x/shape"

//go:tag mkunion:"Schema"
type (
	None   struct{}
	Bool   bool
	Number float64
	String string
	Binary []byte
	List   []Schema
	Map    map[string]Schema
)

//go:tag serde:"json"
type Field struct {
	Name  string
	Value Schema
}

var none = &None{}

func MkNone() *None {
	return none
}

func IsNone(x Schema) bool {
	_, ok := x.(*None)
	return ok
}

func MkBool(b bool) *Bool {
	return (*Bool)(&b)
}

func MkInt(x int64) *Number {
	v := float64(x)
	return (*Number)(&v)
}

func MkUint(x uint64) *Number {
	v := float64(x)
	return (*Number)(&v)
}

func MkFloat(x float64) *Number {
	return (*Number)(&x)
}

func MkBinary(b []byte) *Binary {
	v := Binary(b)
	return &v
}

func MkString(s string) *String {
	return (*String)(&s)
}

func MkList(items ...Schema) *List {
	result := make(List, len(items))
	copy(result, items)
	return &result
}
func MkMap(fields ...Field) *Map {
	var result = make(Map)
	for _, field := range fields {
		result[field.Name] = field.Value
	}
	return &result
}

func MkField(name string, value Schema) Field {
	return Field{
		Name:  name,
		Value: value,
	}
}

func AppendList(list *List, items ...Schema) *List {
	result := append(*list, items...)
	return &result
}

const PktImportName = "github.com/widmogrod/mkunion/x/schema"

var names = map[string]bool{
	"Schema": true,
	"None":   true,
	"Bool":   true,
	"Number": true,
	"String": true,
	"Binary": true,
	"List":   true,
	"Map":    true,
}

func IsShapeASchema(x shape.Shape) bool {
	switch y := x.(type) {
	case *shape.RefName:
		return y.PkgImportName == PktImportName && names[y.Name]
	case *shape.StructLike:
		return y.PkgImportName == PktImportName && names[y.Name]
	case *shape.UnionLike:
		return y.PkgImportName == PktImportName && names[y.Name]
	}

	return false
}
