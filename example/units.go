package example

// --8<-- [start:example]

//go:tag mkunion:"Measurement[Unit]"
type (
	Distance[Unit any] struct{ value float64 }
	Speed[Unit any]    struct{ value float64 }
)

//go:tag mkunion:"Time[Unit]"
type (
	AnyTime[Unit any]      struct{ value float64 }
	PositiveTime[Unit any] struct{ value float64 }
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

// ToFeet Type-safe unit conversions
func (d *Distance[Meters]) ToFeet() *Distance[Feet] {
	return &Distance[Feet]{value: d.value * 3.28084}
}

func (t *PositiveTime[Seconds]) ToHours() *PositiveTime[Hours] {
	return &PositiveTime[Hours]{value: t.value / 3600}
}

func NewTime(value float64) Time[Seconds] {
	if value <= 0 {
		return &AnyTime[Seconds]{value: value}
	}
	return &PositiveTime[Seconds]{value: value}
}

// CalculateSpeed only compatible units can be combined
func CalculateSpeed(distance *Distance[Meters], time *PositiveTime[Seconds]) *Speed[MetersPerSecond] {
	return &Speed[MetersPerSecond]{value: distance.value / time.value}
}

// --8<-- [end:example]
