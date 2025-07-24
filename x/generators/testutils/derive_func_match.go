package testutils

import (
	"bytes"
	"strings"
	"time"
)

//go:tag mkunion:"Alphabet"
type (
	A1 struct{}
	B2 struct{}
	C3 struct{}
)

//go:tag mkunion:"Number"
type (
	N0 struct{}
	N1 struct{}
)

//go:tag mkmatch
type MatchAlphabetNumberTuple[T0 Alphabet, T1 Number] interface {
	Match1(x *A1, y *N0)
	Match2(x *C3, y *time.Duration)
	Match3(x map[Some[*strings.Replacer]]*bytes.Buffer, y *time.Duration)
	Finally(x, y any)
}
