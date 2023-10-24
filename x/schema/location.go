package schema

//go:generate go run ../../cmd/mkunion/main.go -name=Location -skip-extension=schema
type (
	LocationField struct {
		Name string
	}
	LocationIndex struct {
		Index int
	}
	LocationAnything struct{}
)
