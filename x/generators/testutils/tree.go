package testutils

import (
	"github.com/widmogrod/mkunion/x/schema"
	"time"
)

//go:generate go run ../../../cmd/mkunion/main.go

//go:tag mkunion:"Tree"
type (
	Branch struct {
		Lit    Tree
		List   []Tree
		Map    map[string]Tree
		Of     *ListOf[Tree] `json:"just_of"`
		L      *Leaf
		Kattr  [2]*Leaf
		IntPtr *int64
	}
	Leaf struct{ Value int64 }
	K    string
	P    ListOf2[ListOf[any], *ListOf2[int64, *time.Duration]]
	Ma   map[string]Tree
	La   []Tree
	Ka   []map[string]Tree
)

//go:tag mkunion:"Forest"
type (
	Tree2 = Branch
	Leaf2 = Leaf
)

type ListOfAliasAny = ListOf[any]

//go:tag serde:"json"
type ListOf[T any] struct {
	Data T
}

//go:tag serde:"json"
type ListOf2[T1 comparable, T2 any] struct {
	ID        string
	Data      T1
	List      []T2
	Map       map[T1]T2 `json:"map_of_tree"`
	ListOf    ListOf[T1]
	ListOfPtr *ListOf[T2]
	Time      time.Time
	Value     schema.Schema
}
