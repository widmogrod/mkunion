//go:tag mkunion:",no-type-registry"
package testasset

//go:tag mkunion:"SomeDSL"
type (
	Explain struct {
		Example Example `json:"example"`
	}
)
