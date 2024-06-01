package ast

//go:tag mkunion:"SyntaxSugar"
type (
	EqTo     struct{ V any }
	GrThan   struct{ V any }
	OrFields struct{ M map[string]SyntaxSugar }
)
