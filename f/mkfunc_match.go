// Code generated by mkfunc. DO NOT EDIT.
package f

import (
	"errors"
	"fmt"
)

func Match2[TIn, TOut, T1, T2 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	}

	return df(x)
}

func MustMatch2[TIn, TOut, T1, T2 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
) TOut {
	return Match2(x, f1, f2, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T)", x, t1, t2)))
	})
}

func Match3[TIn, TOut, T1, T2, T3 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	}

	return df(x)
}

func MustMatch3[TIn, TOut, T1, T2, T3 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
) TOut {
	return Match3(x, f1, f2, f3, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T)", x, t1, t2, t3)))
	})
}

func Match4[TIn, TOut, T1, T2, T3, T4 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	}

	return df(x)
}

func MustMatch4[TIn, TOut, T1, T2, T3, T4 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
) TOut {
	return Match4(x, f1, f2, f3, f4, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T)", x, t1, t2, t3, t4)))
	})
}

func Match5[TIn, TOut, T1, T2, T3, T4, T5 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	}

	return df(x)
}

func MustMatch5[TIn, TOut, T1, T2, T3, T4, T5 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
) TOut {
	return Match5(x, f1, f2, f3, f4, f5, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5)))
	})
}

func Match6[TIn, TOut, T1, T2, T3, T4, T5, T6 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	}

	return df(x)
}

func MustMatch6[TIn, TOut, T1, T2, T3, T4, T5, T6 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
) TOut {
	return Match6(x, f1, f2, f3, f4, f5, f6, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6)))
	})
}

func Match7[TIn, TOut, T1, T2, T3, T4, T5, T6, T7 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	}

	return df(x)
}

func MustMatch7[TIn, TOut, T1, T2, T3, T4, T5, T6, T7 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
) TOut {
	return Match7(x, f1, f2, f3, f4, f5, f6, f7, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7)))
	})
}

func Match8[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	}

	return df(x)
}

func MustMatch8[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
) TOut {
	return Match8(x, f1, f2, f3, f4, f5, f6, f7, f8, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8)))
	})
}

func Match9[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	}

	return df(x)
}

func MustMatch9[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
) TOut {
	return Match9(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9)))
	})
}

func Match10[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	}

	return df(x)
}

func MustMatch10[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
) TOut {
	return Match10(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10)))
	})
}

func Match11[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	}

	return df(x)
}

func MustMatch11[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
) TOut {
	return Match11(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11)))
	})
}

func Match12[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	}

	return df(x)
}

func MustMatch12[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
) TOut {
	return Match12(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12)))
	})
}

func Match13[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	}

	return df(x)
}

func MustMatch13[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
) TOut {
	return Match13(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13)))
	})
}

func Match14[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	case T14:
		return f14(y)
	}

	return df(x)
}

func MustMatch14[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
) TOut {
	return Match14(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		var t14 T14
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14)))
	})
}

func Match15[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	case T14:
		return f14(y)
	case T15:
		return f15(y)
	}

	return df(x)
}

func MustMatch15[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
) TOut {
	return Match15(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, f15, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		var t14 T14
		var t15 T15
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14, t15)))
	})
}

func Match16[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	case T14:
		return f14(y)
	case T15:
		return f15(y)
	case T16:
		return f16(y)
	}

	return df(x)
}

func MustMatch16[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
) TOut {
	return Match16(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, f15, f16, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		var t14 T14
		var t15 T15
		var t16 T16
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14, t15, t16)))
	})
}

func Match17[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	case T14:
		return f14(y)
	case T15:
		return f15(y)
	case T16:
		return f16(y)
	case T17:
		return f17(y)
	}

	return df(x)
}

func MustMatch17[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
) TOut {
	return Match17(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, f15, f16, f17, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		var t14 T14
		var t15 T15
		var t16 T16
		var t17 T17
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14, t15, t16, t17)))
	})
}

func Match18[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17, T18 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
	f18 func(x T18) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	case T14:
		return f14(y)
	case T15:
		return f15(y)
	case T16:
		return f16(y)
	case T17:
		return f17(y)
	case T18:
		return f18(y)
	}

	return df(x)
}

func MustMatch18[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17, T18 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
	f18 func(x T18) TOut,
) TOut {
	return Match18(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, f15, f16, f17, f18, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		var t14 T14
		var t15 T15
		var t16 T16
		var t17 T17
		var t18 T18
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14, t15, t16, t17, t18)))
	})
}

func Match19[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17, T18, T19 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
	f18 func(x T18) TOut,
	f19 func(x T19) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	case T14:
		return f14(y)
	case T15:
		return f15(y)
	case T16:
		return f16(y)
	case T17:
		return f17(y)
	case T18:
		return f18(y)
	case T19:
		return f19(y)
	}

	return df(x)
}

func MustMatch19[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17, T18, T19 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
	f18 func(x T18) TOut,
	f19 func(x T19) TOut,
) TOut {
	return Match19(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, f15, f16, f17, f18, f19, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		var t14 T14
		var t15 T15
		var t16 T16
		var t17 T17
		var t18 T18
		var t19 T19
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14, t15, t16, t17, t18, t19)))
	})
}

func Match20[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17, T18, T19, T20 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
	f18 func(x T18) TOut,
	f19 func(x T19) TOut,
	f20 func(x T20) TOut,
	df func(x TIn) TOut,
) TOut {
	switch y := any(x).(type) {
	case T1:
		return f1(y)
	case T2:
		return f2(y)
	case T3:
		return f3(y)
	case T4:
		return f4(y)
	case T5:
		return f5(y)
	case T6:
		return f6(y)
	case T7:
		return f7(y)
	case T8:
		return f8(y)
	case T9:
		return f9(y)
	case T10:
		return f10(y)
	case T11:
		return f11(y)
	case T12:
		return f12(y)
	case T13:
		return f13(y)
	case T14:
		return f14(y)
	case T15:
		return f15(y)
	case T16:
		return f16(y)
	case T17:
		return f17(y)
	case T18:
		return f18(y)
	case T19:
		return f19(y)
	case T20:
		return f20(y)
	}

	return df(x)
}

func MustMatch20[TIn, TOut, T1, T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13, T14, T15, T16, T17, T18, T19, T20 any](
	x TIn,
	f1 func(x T1) TOut,
	f2 func(x T2) TOut,
	f3 func(x T3) TOut,
	f4 func(x T4) TOut,
	f5 func(x T5) TOut,
	f6 func(x T6) TOut,
	f7 func(x T7) TOut,
	f8 func(x T8) TOut,
	f9 func(x T9) TOut,
	f10 func(x T10) TOut,
	f11 func(x T11) TOut,
	f12 func(x T12) TOut,
	f13 func(x T13) TOut,
	f14 func(x T14) TOut,
	f15 func(x T15) TOut,
	f16 func(x T16) TOut,
	f17 func(x T17) TOut,
	f18 func(x T18) TOut,
	f19 func(x T19) TOut,
	f20 func(x T20) TOut,
) TOut {
	return Match20(x, f1, f2, f3, f4, f5, f6, f7, f8, f9, f10, f11, f12, f13, f14, f15, f16, f17, f18, f19, f20, func(x TIn) TOut {
		var t1 T1
		var t2 T2
		var t3 T3
		var t4 T4
		var t5 T5
		var t6 T6
		var t7 T7
		var t8 T8
		var t9 T9
		var t10 T10
		var t11 T11
		var t12 T12
		var t13 T13
		var t14 T14
		var t15 T15
		var t16 T16
		var t17 T17
		var t18 T18
		var t19 T19
		var t20 T20
		panic(errors.New(fmt.Sprintf("unexpected match type %T. expected (%T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T or %T)", x, t1, t2, t3, t4, t5, t6, t7, t8, t9, t10, t11, t12, t13, t14, t15, t16, t17, t18, t19, t20)))
	})
}
