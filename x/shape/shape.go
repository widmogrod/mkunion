package shape

// go:generate go run ../../cmd/mkunion/main.go -name=Shape
type (
	Any     struct{}
	RefName struct {
		Name          string
		PkgName       string
		PkgImportName string
		IsPointer     bool
		Indexed       []Shape
	}
	AliasLike struct {
		Name          string
		PkgName       string
		PkgImportName string
		IsAlias       bool
		Type          Shape
		Tags          map[string]Tag
	}
	BooleanLike struct{}
	// StringLike is a string type, and when it has name, it means it named type.
	// For example:
	// 	type C string
	StringLike struct {
		//Guard Guard
	}
	NumberLike struct {
		Kind NumberKind
		//Guard Guard
	}
	ListLike struct {
		//Extend *ListLike
		//Guard  Guard
		Element          Shape
		ElementIsPointer bool
		// ArrayLen is a pointer to int, when it's nil, it means it's a slice.
		ArrayLen *int
	}
	MapLike struct {
		//Extend *MapLike
		//Guard  Guard
		Key          Shape
		Val          Shape
		KeyIsPointer bool
		ValIsPointer bool
	}
	StructLike struct {
		Name          string
		PkgName       string
		PkgImportName string
		TypeParams    []TypeParam
		Fields        []*FieldLike
		Tags          map[string]Tag
	}
	UnionLike struct {
		Name          string
		PkgName       string
		PkgImportName string
		Variant       []Shape
		Tags          map[string]Tag
	}
)

type TypeParam struct {
	Name string
	Type Shape
}

// go:generate go run ../../cmd/mkunion/main.go -name=NumberKind
type (
	UInt8   struct{}
	UInt16  struct{}
	UInt32  struct{}
	UInt64  struct{}
	Int8    struct{}
	Int16   struct{}
	Int32   struct{}
	Int64   struct{}
	Float32 struct{}
	Float64 struct{}
)

var TypeStringToNumberKindMap = map[string]NumberKind{
	"uint8":   &UInt8{},
	"uint16":  &UInt16{},
	"uint32":  &UInt32{},
	"uint64":  &UInt64{},
	"int8":    &Int8{},
	"int16":   &Int16{},
	"int32":   &Int32{},
	"int64":   &Int64{},
	"float32": &Float32{},
	"float64": &Float64{},
	"byte":    &UInt8{},
	"rune":    &Int32{},
}

type FieldLike struct {
	Name      string
	Type      Shape
	Desc      *string
	Guard     Guard
	IsPointer bool
	Tags      map[string]Tag
}

type Tag struct {
	Value   string
	Options []string
}

// go:generate go run ../../cmd/mkunion/main.go -name=Guard
type (
	Enum struct {
		Val []string
	}
	Required struct{}
	//Regexp   struct {
	//	Regexp string
	//}
	//Between struct {
	//	Min int
	//	Max int
	//}
	AndGuard struct {
		L []Guard
	}
)

func ConcatGuard(a, b Guard) Guard {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	var result *AndGuard
	if and, ok := a.(*AndGuard); ok {
		result = and
	} else {
		result = &AndGuard{
			L: []Guard{a},
		}
	}

	return &AndGuard{
		L: append(result.L, b),
	}
}

func Tags(x Shape) map[string]Tag {
	return MustMatchShape(
		x,
		func(x *Any) map[string]Tag {
			return nil
		},
		func(x *RefName) map[string]Tag {
			return nil
		},
		func(x *AliasLike) map[string]Tag {
			return x.Tags
		},
		func(x *BooleanLike) map[string]Tag {
			return nil
		},
		func(x *StringLike) map[string]Tag {
			return nil
		},
		func(x *NumberLike) map[string]Tag {
			return nil
		},
		func(x *ListLike) map[string]Tag {
			return nil
		},
		func(x *MapLike) map[string]Tag {
			return nil
		},
		func(x *StructLike) map[string]Tag {
			return x.Tags
		},
		func(x *UnionLike) map[string]Tag {
			return x.Tags
		},
	)
}
