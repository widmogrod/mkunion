package ast

//go:generate go run ../../cmd/mkunion/main.go -name=SyntaxSugar -types=EqTo,GrThan,OrFields -skip-extension=schema
type (
	EqTo     struct{ V any }
	GrThan   struct{ V any }
	OrFields map[string]SyntaxSugar
)
