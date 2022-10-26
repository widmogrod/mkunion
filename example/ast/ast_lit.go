package ast

//go:generate go run ../../cmd/mkunion/main.go -name=Value -types=Lit,Accessor
type (
	Lit      struct{ Value any }
	Accessor struct{ Path []string }
)
