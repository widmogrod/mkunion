package ast

//go:generate go run ../../cmd/mkunion/main.go -name=Value -types=ALit,AAccessor
type (
	ALit      struct{ Value any }
	AAccessor struct{ Path []string }
)
