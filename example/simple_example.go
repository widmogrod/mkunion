package example

//go:generate go run ../cmd/mkunion/main.go golang -name=Vehicle -types=Plane,Car,Boat
type (
	Car   struct{}
	Plane struct{}
	Boat  struct{}
)
