//go:tag mkunion:",no-type-registry"
package testasset

import (
	"time"
)

//go:tag mkunion:"Example"
type (
	A struct {
		Name string `json:"name" desc:"Name of the person"`
	}
	B struct {
		Age int `json:"age"`
		A   *A
		T   *time.Time
	}
	C string
	D int64
	E float64
	F bool
	H map[string]Example
	I []Example
	J [2]string
	K A
	L = List
	//go:tag json:"m_list,omitempty"
	M List
	N time.Duration
	O ListOf[time.Duration]
	P ListOf2[ListOf[any], *ListOf2[int64, *time.Duration]] // √ - found by FindInstantiationsOf 2x
)

//go:tag mkunion:"AliasExample"
type (
	A2 = A
	B2 = B
)

// List is a list of elements
//
//go:tag json:"list,omitempty" serde:"json"
type List struct{}

//go:tag serde:"json" json:"list_of,omitempty"
type ListOf[T any] struct{}

// ListOf2 is a list of 2 elements
//
//go:tag serde:"json"
//go:tag json:"list_of_2,omitempty"
type ListOf2[T1, T2 any] struct {
	Data   T1
	ListOf ListOf[T1] `json:"list_of"`
}

func init() {
	_ = &ListOf2[*float64, *ListOf2[*A2, *time.Ticker]]{} // √ - found by FindInstantiationsOf 2x
	_ = func(_ *ListOf2[*B2, time.Month]) {}              // √ - found by FindInstantiationsOf
	var _ ListOf2[*F, float64]                            // √ - found by FindInstantiationsOf
}

type _someInterface interface {
	Do(*ListOf2[*O, time.Location]) // √ - found by FindInstantiationsOf
}

type _someStruct struct {
	B *ListOf2[*K, time.Weekday] // √ - found by FindInstantiationsOf
}

func (*_someStruct) Exec(*ListOf2[*L, time.Location]) {} // √ - found by FindInstantiationsOf

var (
	_ = &ListOf2[ListOf[*bool], *ListOf2[Example, *time.Time]]{} // √ - found by FindInstantiationsOf x2
)

//go:tag mkunion:"Option[AZ]"
type (
	Some[AZ ListOf2[*O, time.Location]] struct {
		Data AZ
	}
	None[AZ ListOf2[*O, time.Location]] struct{}
)

//go:tag mkmatch:"MyMatchWithCustomName"
type MyMach[A, B Example] interface {
	MatchEs(a, b E)
	Finally(a, b Example)
}
