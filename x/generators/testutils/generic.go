package testutils

//go:generate go run ../../../cmd/mkunion

//go:tag mkunion:"Record,noserde"
type (
	Item[A any] struct {
		Key  string
		Data A
	}

	Other[A any] Some[A]
)

type Some[B any] struct {
	ValueOf B
}
