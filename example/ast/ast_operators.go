package ast

//go:generate go run ../../cmd/mkunion/main.go -name=Operator -types=Eq,Gt,Or
type (
	Eq struct{ L, R Value }
	Gt struct{ L, R Value }
	Or []Operator
)
