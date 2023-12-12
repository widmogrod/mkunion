package testasset

// go:generate go run ../../../cmd/mkunion/main.go --name=GraphDSL
type (
	Graph[T any] struct {
		Vertices map[string]*Vertex[T]
	}
	Vertex[T any] struct {
		Value T
		Edges []*Edge[T]
	}
	Edge[T any] struct {
		Weight float64
	}
)
