package example

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:generate go run ../cmd/mkunion/main.go golang -name=Vehicle -types=Plane,Car,Boat -output=simple_union_example_gen_test.go -package=example
type (
	Car   struct{}
	Plane struct{}
	Boat  struct{}
)

type vehiclePrinter struct{}

func (t *vehiclePrinter) VisitPlane(v *Plane) any { return fmt.Sprintf("Plane") }
func (t *vehiclePrinter) VisitCar(v *Car) any     { return fmt.Sprintf("Car") }
func (t *vehiclePrinter) VisitBoat(v *Boat) any   { return fmt.Sprintf("Boat") }

var _ VehicleVisitor = (*vehiclePrinter)(nil)

func TestGeneratedVisitor(t *testing.T) {
	car := &Car{}
	plane := &Plane{}
	boat := &Boat{}

	visitor := &vehiclePrinter{}
	assert.Equal(t, "Car", car.Accept(visitor))
	assert.Equal(t, "Plane", plane.Accept(visitor))
	assert.Equal(t, "Boat", boat.Accept(visitor))
}
