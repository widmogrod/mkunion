package example

//go:generate go run ../cmd/mkunion/main.go -name=Vehicle -types=Plane,Car,Boat -path=simple_union_example_gen_test -packageName=example
type (
	Car   struct{}
	Plane struct{}
	Boat  struct{}
)
