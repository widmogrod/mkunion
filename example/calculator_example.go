package example

//go:generate go run ../cmd/mkunion/main.go --name=Calc
type (
	Lit struct{ V int }
	Sum struct{ Left, Right Calc }
	Mul struct{ Left, Right Calc }
)
