package testasset

//go:generate go run ../../../cmd/mkunion/main.go

//go:tag mkunion:"SomeDSL"
type (
	Explain struct {
		Example Example `json:"example"`
	}
)
