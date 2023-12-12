package testutils

//go:generate go run ../../../cmd/mkunion/main.go -name=Tree -include-extension=reducer_bfs,reducer_dfs,default_visitor,default_reducer
type (
	Branch struct {
		Lit  Tree
		List []Tree
		Map  map[string]Tree
	}
	Leaf struct{ Value int64 }
	K    string
)
