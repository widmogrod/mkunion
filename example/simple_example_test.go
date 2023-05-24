package example

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert.Equal(t, "Car", car.AcceptVehicle(visitor))
	assert.Equal(t, "Plane", plane.AcceptVehicle(visitor))
	assert.Equal(t, "Boat", boat.AcceptVehicle(visitor))
}
