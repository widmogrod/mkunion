package shape

// go:generate go run ../../cmd/mkunion/main.go -name=Shape
type (
	Any     struct{}
	RefName struct {
		Name          string
		PkgName       string
		PkgImportName string
	}
	BooleanLike struct{}
	StringLike  struct {
		//Guard Guard
	}
	NumberLike struct {
		//Guard Guard
	}
	ListLike struct {
		//Extend *ListLike
		//Guard  Guard
		Element          Shape
		ElementIsPointer bool
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
		Fields        []*FieldLike
	}
	UnionLike struct {
		Name          string
		PkgName       string
		PkgImportName string
		Variant       []Shape
	}
)

type FieldLike struct {
	Name      string
	Type      Shape
	Desc      *string
	Guard     Guard
	IsPointer bool
	Tags      map[string]FieldTag
}

type FieldTag struct {
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
