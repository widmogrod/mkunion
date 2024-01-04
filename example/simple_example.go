package example

//go:generate go run ../cmd/mkunion/main.go

//go:tag mkunion:"Vehicle"
type (
	Car   struct{}
	Plane struct{}
	Boat  struct{}
)
