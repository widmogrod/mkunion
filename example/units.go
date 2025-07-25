package example

// --8<-- [start:example]

//go:tag mkunion:"Measurement[Unit]"
type (
	Distance[Unit any] struct{ value float64 }
	Time[Unit any]     struct{ value float64 }
	Speed[Unit any]    struct{ value float64 }
)

type Meters struct{}
type Feet struct{}
type Seconds struct{}
type Hours struct{}
type MetersPerSecond struct{}
type MilesPerHour struct{}

func NewDistance(value float64) *Distance[Meters] {
	return &Distance[Meters]{value: value}
}

func NewTime(value float64) *Time[Seconds] {
	return &Time[Seconds]{value: value}
}

// ToFeet Type-safe unit conversions
func (d *Distance[Meters]) ToFeet() *Distance[Feet] {
	return &Distance[Feet]{value: d.value * 3.28084}
}

func (t *Time[Seconds]) ToHours() *Time[Hours] {
	return &Time[Hours]{value: t.value / 3600}
}

// CalculateSpeed only compatible units can be combined
func CalculateSpeed(distance *Distance[Meters], time *Time[Seconds]) *Speed[MetersPerSecond] {
	return &Speed[MetersPerSecond]{value: distance.value / time.value}
}

// --8<-- [end:example]
