package example

import "fmt"

// --8<-- [start:vehicle-def]

//go:tag mkunion:"Vehicle"
type (
	Car struct {
		Color  string
		Wheels int
	}
	Plane struct {
		Color   string
		Engines int
	}
	Boat struct {
		Color      string
		Propellers int
	}
)

// --8<-- [end:vehicle-def]

// --8<-- [start:calculate-fuel]

func CalculateFuelUsage(v Vehicle) int {
	return MatchVehicleR1(
		v,
		func(x *Car) int {
			return x.Wheels * 2
		},
		func(x *Plane) int {
			return x.Engines * 10
		},
		func(x *Boat) int {
			return x.Propellers * 5
		},
	)
}

// --8<-- [end:calculate-fuel]

// --8<-- [start:match-def]

//go:tag mkmatch:"MatchPairs"
type MatchPairs[A, B Vehicle] interface {
	MatchCars(x, y *Car)
	MatchBoatAny(x *Boat, y any)
	Finally(x, y any)
}

// --8<-- [end:match-def]

// --8<-- [start:match-pairs]

func NamePairs(x, y Vehicle) string {
	return MatchPairsR1(
		x, y,
		func(x0 *Car, x1 *Car) string {
			return fmt.Sprintf("Car %s vs Car %s", x0.Color, x1.Color)
		},
		func(x0 *Boat, x1 any) string {
			return fmt.Sprintf("Boat %s vs %T", x0.Color, x1)
		},
		func(x0 any, x1 any) string {
			return fmt.Sprintf("Finally")
		},
	)
}

// --8<-- [end:match-pairs]
