package shape

//go:generate go run ../../cmd/mkunion/main.go -name=Shape
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
		Element Shape
	}
	MapLike struct {
		//Extend *MapLike
		//Guard  Guard
		Key Shape
		Val Shape
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
		Variant       []*StructLike
	}
)

type FieldLike struct {
	Name string
	Type Shape
}

//go:generate go run ../../cmd/mkunion/main.go -name=Guard
type (
	Regexp struct {
		Regexp string
	}
	Between struct {
		Min int
		Max int
	}
	AndGuard struct {
		L []Guard
	}
	OrGuard struct {
		L []Guard
	}
)
