package ast

//go:generate go run ../../cmd/mkunion/main.go -name=Operator -types=AEq,AGt,AOr
type (
	AEq struct{ L, R Value }
	AGt struct{ L, R Value }
	AOr []Operator
)
