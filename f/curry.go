package f

func Curry2[A, B, C any](f func(A, B) C) func(A) func(B) C {
	return func(x A) func(B) C {
		return func(y B) C {
			return f(x, y)
		}
	}
}
