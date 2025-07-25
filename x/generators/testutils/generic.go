package testutils

//go:tag mkunion:"Record[A]"
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
