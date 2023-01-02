package f

func Partial2[A, B, C any](f func(A, B) C, x A) func(B) C {
	return func(y B) C {
		return f(x, y)
	}
}
