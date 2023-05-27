package example

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

//	func MatchMR0[T0 Alphabet, T1 Number](a T0, b T1, m MatchAlphabetNumberTuple[T0, T1]) {
//		panic(" is not exhaustive")
//	}
//
// type (
//
//	MMatch1[T0 Alphabet, T1 Number] func(x *A, y *N0)
//	MMatch2[T0 Alphabet, T1 Number] func(x *C, y any)
//	MMatch3[T0 Alphabet, T1 Number] func(x, y any)
//
// )
func TestDeriveFunctionInAction(t *testing.T) {
	useCases := map[string]struct {
		x              Alphabet
		y              Number
		expectedCallNo int
	}{
		"should match first case": {
			x:              &A1{},
			y:              &N0{},
			expectedCallNo: 1,
		},
		"should match second case when second argument N0": {
			x:              &C3{},
			y:              &N0{},
			expectedCallNo: 2,
		},
		"should match second case when second argument N1": {
			x:              &C3{},
			y:              &N1{},
			expectedCallNo: 2,
		},
		"should match third case": {
			x:              &B2{},
			y:              &N1{},
			expectedCallNo: 3,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			MatchAlphabetNumberTupleR0[Alphabet, Number](
				uc.x, uc.y,
				func(x *A1, y *N0) {
					assert.Equal(t, uc.expectedCallNo, 1)
				},
				func(x *C3, y any) {
					assert.Equal(t, uc.expectedCallNo, 2)
				},
				func(x, y any) {
					assert.Equal(t, uc.expectedCallNo, 3)
				},
			)

			res1 := MatchAlphabetNumberTupleR1[Alphabet, Number](
				uc.x, uc.y,
				func(x *A1, y *N0) int {
					assert.Equal(t, uc.expectedCallNo, 1)
					return 1
				},
				func(x *C3, y any) int {
					assert.Equal(t, uc.expectedCallNo, 2)
					return 2
				},
				func(x, y any) int {
					assert.Equal(t, uc.expectedCallNo, 3)
					return 3
				},
			)
			assert.Equal(t, uc.expectedCallNo, res1)

			res2a, res2b := MatchAlphabetNumberTupleR2[Alphabet, Number](
				uc.x, uc.y,
				func(x *A1, y *N0) (int, string) {
					assert.Equal(t, uc.expectedCallNo, 1)
					return 1, strconv.Itoa(1)
				},
				func(x *C3, y any) (int, string) {
					assert.Equal(t, uc.expectedCallNo, 2)
					return 2, strconv.Itoa(2)
				},
				func(x, y any) (int, string) {
					assert.Equal(t, uc.expectedCallNo, 3)
					return 3, strconv.Itoa(3)
				},
			)
			assert.Equal(t, uc.expectedCallNo, res2a)
			assert.Equal(t, strconv.Itoa(uc.expectedCallNo), res2b)
		})
	}
}
