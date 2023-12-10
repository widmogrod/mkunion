package testutils

//go:generate go run ../../../cmd/mkunion/main.go -name=Tree
type (
	Branch struct{ L, R Tree }
	Leaf   struct{ Value int }
)
