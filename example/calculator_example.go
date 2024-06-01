package example

//go:tag mkunion:"Calc"
type (
	Lit struct{ V int }
	Sum struct{ Left, Right Calc }
	Mul struct{ Left, Right Calc }
)
