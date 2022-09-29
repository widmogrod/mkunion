package example

//go:generate mkunion --name=Calc --types=Lit,Sum,Mul
type (
	// Calculator is a calculator.
	Lit struct{ V int }
	Sum struct{ Left, Right Calc }
	Mul struct{ Left, Right Calc }
)
