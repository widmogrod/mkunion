package example

//go:generate go run ../cmd/mkunion/main.go -name=Alphabet
type (
	A1 struct{}
	B2 struct{}
	C3 struct{}
)

//go:generate go run ../cmd/mkunion/main.go -name=Number
type (
	N0 struct{}
	N1 struct{}
)

//go:generate go run ../cmd/mkunion/main.go match -name=MatchAlphabetNumberTuple
type MatchAlphabetNumberTuple[T0 Alphabet, T1 Number] interface {
	Match1(x *A1, y *N0)
	Match2(x *C3, y any)
	Match3(x, y any)
}
