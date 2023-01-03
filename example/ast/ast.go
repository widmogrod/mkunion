package ast

//go:generate go run ../../cmd/mkunion/main.go -name=Value -types=Lit,Accessor
type (
	Lit      struct{ Value any }
	Accessor struct{ Path []string }
)

//go:generate go run ../../cmd/mkunion/main.go -name=Operator -types=Eq,Gt,Or,And,Not
type (
	Eq  struct{ L, R Value }
	Gt  struct{ L, R Value }
	Or  []Operator
	And []Operator
	Not struct{ Operator }
)
