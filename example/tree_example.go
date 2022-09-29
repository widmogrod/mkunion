package example

//go:generate go run ../cmd/mkunion/main.go -name=Tree -types=Branch,Leaf
type (
	Branch struct{ L, R Tree }
	Leaf   struct{ Value int }
)
