package example

//go:generate go run ../cmd/mkunion/main.go -name=Tree2
type (
	Branch2 struct {
		Lit  Tree2
		List []Tree2
		Map  map[string]Tree2
	}
	Leaf2 struct{ Value int }
)
