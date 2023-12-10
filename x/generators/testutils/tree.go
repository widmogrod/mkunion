package testutils

//go:generate go run ../../../cmd/mkunion/main.go -name=Tree
type (
	Branch struct {
		Lit  Tree
		List []Tree
		Map  map[string]Tree
	}
	Leaf struct{ Value int }
)
