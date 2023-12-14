package testasset

import "time"

//go:generate go run ../../../cmd/mkunion/main.go --name=Example
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
	// L Example is not allowed, since Example is interface,
	// and interface cannot have methods implemented as Visitor pattern requires
	L = List
	M List
	O time.Duration
)

type List struct{}
