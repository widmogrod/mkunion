package example

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:generate go run ../cmd/mkunion/main.go -name=Vehicle -types=Plane,Car,Boat -path=visitor_example_visitor_test -packageName=example
type (
	Car   struct{}
	Plane struct{}
	Boat  struct{}
)

//go:generate go run ../cmd/mkunion/main.go -name=WherePredicate -types=Eq,Gt,Lt,And,Or,Path -path=where_predicate_example_gen_test -packageName=example
type (
	Eq   struct{ V interface{} }
	Gt   struct{ V interface{} }
	Lt   struct{ V interface{} }
	And  []WherePredicate
	Or   []WherePredicate
	Path struct {
		Parts     []string
		Condition WherePredicate
	}
)

type testVisitor struct{}

func (t *testVisitor) VisitPlane(v *Plane) any { return fmt.Sprintf("Plane") }
func (t *testVisitor) VisitCar(v *Car) any     { return fmt.Sprintf("Car") }
func (t *testVisitor) VisitBoat(v *Boat) any   { return fmt.Sprintf("Boat") }

var _ VehicleVisitor = (*testVisitor)(nil)

func TestGeneratedVisitor(t *testing.T) {
	car := &Car{}
	plane := &Plane{}
	boat := &Boat{}

	visitor := &testVisitor{}
	assert.Equal(t, "Car", car.Accept(visitor))
	assert.Equal(t, "Plane", plane.Accept(visitor))
	assert.Equal(t, "Boat", boat.Accept(visitor))
}

func TestPredicate(t *testing.T) {
	_ = And{
		&Path{
			Parts:     []string{"name"},
			Condition: &Eq{"bar"},
		},
	}

	//visitor := &testVisitor{}
	//assert.Equal(t, "And", predicate.Accept(visitor))
}
