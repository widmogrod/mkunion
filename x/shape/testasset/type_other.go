package testasset

//go:generate go run ../../../cmd/mkunion/main.go -name=SomeDSL
type (
	Explain struct {
		Example Example `json:"example"`
	}
)
