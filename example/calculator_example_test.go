package example

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestCalculatorExample(t *testing.T) {
	// (2 + (2 * 3))
	var calculation Calc = &Sum{
		Left: &Lit{2},
		Right: &Mul{
			Left:  &Lit{2},
			Right: &Lit{3},
		},
	}

	result := calculation.Accept(&calculator{}).(int)
	assert.Equal(t, 8, result)

	str := calculation.Accept(&calculatorPrint{}).(string)
	fmt.Println(str)
}

func TestCalculatorDynamicExample(t *testing.T) {
	for i := 0; i < 5; i++ {
		expect := rand.Intn(1000)
		calculation := GenerateCalcExpressions(expect)
		str := calculation.Accept(&calculatorPrint{}).(string)
		t.Logf("expressions: %d = %s", expect, str)
		result := calculation.Accept(&calculator{}).(int)
		assert.Equal(t, expect, result)
	}
}

func GenerateCalcExpressions(x int) Calc {
	if rand.Float64() < 0.3 {
		return &Lit{x}
	}

	if x%3 == 0 {
		return &Mul{
			Left:  GenerateCalcExpressions(3),
			Right: GenerateCalcExpressions(x / 3),
		}
	}

	if x > 10 {
		i := rand.Int() % 10
		return &Sum{
			Left:  GenerateCalcExpressions(i),
			Right: GenerateCalcExpressions(x - i),
		}
	}

	return &Lit{x}
}

var _ CalcVisitor = (*calculator)(nil)

type calculator struct{}

func (c *calculator) VisitLit(v *Lit) any {
	return v.V
}

func (c *calculator) VisitSum(v *Sum) any {
	return v.Left.Accept(c).(int) + v.Right.Accept(c).(int)
}

func (c *calculator) VisitMul(v *Mul) any {
	return v.Left.Accept(c).(int) * v.Right.Accept(c).(int)
}

var _ CalcVisitor = (*calculatorPrint)(nil)

type calculatorPrint struct{}

func (c *calculatorPrint) VisitLit(v *Lit) any {
	return fmt.Sprintf("%d", v.V)
}

func (c *calculatorPrint) VisitSum(v *Sum) any {
	return fmt.Sprintf("(%s + %s)", v.Left.Accept(c).(string), v.Right.Accept(c).(string))
}

func (c *calculatorPrint) VisitMul(v *Mul) any {
	return fmt.Sprintf("(%s * %s)", v.Left.Accept(c).(string), v.Right.Accept(c).(string))
}
