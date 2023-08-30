package shape

//go:generate go run ../../cmd/mkunion/main.go -name=Shape
type (
	BooleanLike struct{}
	StringLike  struct {
		Guard Guard
	}
	NumberLike struct {
		Guard Guard
	}
	ListLike struct {
		Extend *ListLike
		Guard  Guard
		Items  []Shape
	}
	MapLike struct {
		Extend *MapLike
		Guard  Guard
		Field  []FieldLike
	}
)

type FieldLike struct {
	Name  StringLike
	Shape Shape
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
