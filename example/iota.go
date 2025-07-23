package example

import "math"

// --8<-- [start:example]

type ShapeType int

const (
	CircleType ShapeType = iota
	RectangleType
	TriangleType
)

type (
	Circle    struct{ Radius float64 }
	Rectangle struct{ Width, Height float64 }
	Triangle  struct{ Base, Height float64 }
)

type Shape struct {
	Type     ShapeType
	Circle   *Circle
	Rect     *Rectangle
	Triangle *Triangle
}

func CalculateArea(s Shape) float64 {
	switch s.Type {
	case CircleType:
		if s.Circle != nil {
			return math.Pi * s.Circle.Radius * s.Circle.Radius
		}
	case RectangleType:
		if s.Rect != nil {
			return s.Rect.Width * s.Rect.Height
		}
		// Missing TriangleType will not tell you that it's missing switch case
		// you need to use tools like https://github.com/nishanths/exhaustive
	}
	return 0
}

// --8<-- [end:example]
