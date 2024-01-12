package testutils

//go:generate go run ../../../cmd/mkunion

//go:tag mkunion:"Record,noserde"
type (
	Item[A any] struct {
		Key  string
		Data A
	}

	//Error[A any] struct {
	//	Err A
	//}
)
