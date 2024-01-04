package example

//go:generate go run ../cmd/mkunion/main.go

//go:tag mkunion:"Tree"
type (
	Branch struct{ L, R Tree }
	Leaf   struct{ Value int }
)

//go:generate go run ../cmd/mkunion/main.go match -name=MyTriesMatch
type MyTriesMatch[T0, T1 Tree] interface {
	MatchLeafs(*Leaf, *Leaf)
	MatchBranches(*Branch, any)
	MatchMixed(any, any)
}
