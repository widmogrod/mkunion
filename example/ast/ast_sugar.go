package ast

//go:generate go run ../../cmd/mkunion/main.go -name=SyntaxSugar -skip-extension=schema
type (
	EqTo     struct{ V any }
	GrThan   struct{ V any }
	OrFields struct{ M map[string]SyntaxSugar }
)
