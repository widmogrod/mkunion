package testasset

//go:generate go run ../../../cmd/mkunion/main.go --type-registry=false

//go:tag mkunion:"SomeDSL"
type (
	Explain struct {
		Example Example `json:"example"`
	}
)
