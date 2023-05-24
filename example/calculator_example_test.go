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

	result := calculation.AcceptCalc(&calculator{}).(int)
	assert.Equal(t, 8, result)

	str := calculation.AcceptCalc(&calculatorPrint{}).(string)
	fmt.Println(str)

	result = Cal(calculation)
	assert.Equal(t, 8, result)
}

func Cal(x Calc) int {
	return MustMatchCalc(x, func(x *Lit) int {
		return x.V
	}, func(x *Sum) int {
		return Cal(x.Left) + Cal(x.Right)
	}, func(x *Mul) int {
		return Cal(x.Left) * Cal(x.Right)
	})
}

func TestCalculatorDynamicExample(t *testing.T) {
	for i := 0; i < 5; i++ {
		expect := rand.Intn(1000)
		calculation := GenerateCalcExpressions(expect)
		str := calculation.AcceptCalc(&calculatorPrint{}).(string)
		t.Logf("expressions: %d = %s", expect, str)
		result := calculation.AcceptCalc(&calculator{}).(int)
		assert.Equal(t, expect, result)
		result = Cal(calculation)
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
	return v.Left.AcceptCalc(c).(int) + v.Right.AcceptCalc(c).(int)
}

func (c *calculator) VisitMul(v *Mul) any {
	return v.Left.AcceptCalc(c).(int) * v.Right.AcceptCalc(c).(int)
}

var _ CalcVisitor = (*calculatorPrint)(nil)

type calculatorPrint struct{}

func (c *calculatorPrint) VisitLit(v *Lit) any {
	return fmt.Sprintf("%d", v.V)
}

func (c *calculatorPrint) VisitSum(v *Sum) any {
	return fmt.Sprintf("(%s + %s)", v.Left.AcceptCalc(c).(string), v.Right.AcceptCalc(c).(string))
}

func (c *calculatorPrint) VisitMul(v *Mul) any {
	return fmt.Sprintf("(%s * %s)", v.Left.AcceptCalc(c).(string), v.Right.AcceptCalc(c).(string))
}

/*
Benchmark show that function is ~1.5x faster than visitor pattern!
BenchmarkCalcVisitor-8          10280121               116.0 ns/op            56 B/op          7 allocs/op
BenchmarkCalcVisitor-8          10285610               115.4 ns/op            56 B/op          7 allocs/op
BenchmarkCalcVisitor-8          10358955               116.3 ns/op            56 B/op          7 allocs/op
BenchmarkCalcVisitor-8          10388298               122.7 ns/op            56 B/op          7 allocs/op
BenchmarkCalcVisitor-8          10213692               116.5 ns/op            56 B/op          7 allocs/op
BenchmarkCakFunction-8          13601168                89.85 ns/op            0 B/op          0 allocs/op
BenchmarkCakFunction-8          13480336                89.31 ns/op            0 B/op          0 allocs/op
BenchmarkCakFunction-8          13494511                88.60 ns/op            0 B/op          0 allocs/op
BenchmarkCakFunction-8          13612425                88.93 ns/op            0 B/op          0 allocs/op
BenchmarkCakFunction-8          13485291                89.75 ns/op            0 B/op          0 allocs/op
*/
var (
	benchCalcResult      int
	benchCalcExpect      = 10000
	benchCalcCalculation = GenerateCalcExpressions(benchCalcExpect)
)

func BenchmarkCalcVisitor(b *testing.B) {
	var r int
	for i := 0; i < b.N; i++ {
		r = benchCalcCalculation.AcceptCalc(&calculator{}).(int)
		if r != benchCalcExpect {
			b.Fatalf("expect %d, got %d", benchCalcExpect, r)
		}
	}
	benchCalcResult = r
}

func BenchmarkCakFunction(b *testing.B) {
	var r int
	for i := 0; i < b.N; i++ {
		r = Cal(benchCalcCalculation)
		if r != benchCalcExpect {
			b.Fatalf("expect %d, got %d", benchCalcExpect, r)
		}
	}
	benchCalcResult = r
}
