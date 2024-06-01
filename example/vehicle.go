package example

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
