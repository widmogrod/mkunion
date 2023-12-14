package testutils

import "time"

//go:generate go run ../../../cmd/mkunion/main.go -name=Tree -include-extension=reducer_bfs,reducer_dfs,default_visitor,default_reducer
type (
	Branch struct {
		Lit  Tree
		List []Tree
		Map  map[string]Tree
	}
	Leaf struct{ Value int64 }
	K    string
	P    ListOf2[ListOf[any], *ListOf2[int64, *time.Duration]]
)

type ListOf[T any] struct{}
type ListOf2[T1, T2 any] struct{}
