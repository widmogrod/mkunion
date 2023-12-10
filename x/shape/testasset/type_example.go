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
)
