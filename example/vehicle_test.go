package example

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shared"
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

// --8<-- [start:json]

func ExampleVehicleToJSON() {
	var vehicle Vehicle = &Car{
		Color:  "black",
		Wheels: 4,
	}
	result, _ := shared.JSONMarshal[Vehicle](vehicle)
	fmt.Println(string(result))
	// Output: {"$type":"example.Car","example.Car":{"Color":"black","Wheels":4}}
}

func ExampleVehicleFromJSON() {
	input := []byte(`{"$type":"example.Car","example.Car":{"Color":"black","Wheels":4}}`)
	vehicle, _ := shared.JSONUnmarshal[Vehicle](input)
	fmt.Printf("%#v", vehicle)
	// Output: &example.Car{Color:"black", Wheels:4}
}

// --8<-- [end:json]

func ExampleMatchPairs() {
	var vehicle1 Vehicle = &Boat{Color: "black", Propellers: 10}
	var vehicle2 Vehicle = &Plane{Color: "white", Engines: 2}

	result := NamePairs(vehicle1, vehicle2)

	fmt.Println(result)
	// Output: Boat black vs *example.Plane
}
