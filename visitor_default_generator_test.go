package mkunion

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOptionalVisitor(t *testing.T) {
	g := VisitorDefaultGenerator{
		Name:  "Vehicle",
		Types: []string{"Plane", "Car", "Boat"},
	}

	result, err := g.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `

type VehicleNonExhaustiveG[A any] struct {
	Default A
	OnPlane func(x *Plane) A
	OnCar func(x *Car) A
	OnBoat func(x *Boat) A
}
func (t *VehicleNonExhaustiveG[A]) VisitPlane(v *Plane) any {
	if t.OnPlane != nil {
		return t.OnPlane(v)
	}
	return t.Default
}
func (t *VehicleNonExhaustiveG[A]) VisitCar(v *Car) any {
	if t.OnCar != nil {
		return t.OnCar(v)
	}
	return t.Default
}
func (t *VehicleNonExhaustiveG[A]) VisitBoat(v *Boat) any {
	if t.OnBoat != nil {
		return t.OnBoat(v)
	}
	return t.Default
}
`, string(result))
}
