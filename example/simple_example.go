package example

//go:generate go run ../cmd/mkunion/main.go -name=Vehicle
type (
	Car   struct{}
	Plane struct{}
	Boat  struct{}
)
