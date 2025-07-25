package example

import "math"

// --8<-- [start:example]

type IotaShapeType int

const (
	IotaCircleType IotaShapeType = iota
	IotaRectangleType
	IotaTriangleType
)

type (
	IotaCircle    struct{ Radius float64 }
	IotaRectangle struct{ Width, Height float64 }
	IotaTriangle  struct{ Base, Height float64 }
)

type IotaShape struct {
	Type     IotaShapeType
	Circle   *IotaCircle
	Rect     *IotaRectangle
	Triangle *IotaTriangle
}

func CalculateIotaArea(s IotaShape) float64 {
	switch s.Type {
	case IotaCircleType:
		if s.Circle != nil {
			return math.Pi * s.Circle.Radius * s.Circle.Radius
		}
	case IotaRectangleType:
		if s.Rect != nil {
			return s.Rect.Width * s.Rect.Height
		}
		// Missing IotaTriangleType will not tell you that it's missing switch case
		// you need to use tools like https://github.com/nishanths/exhaustive
	}
	return 0
}

// --8<-- [end:example]
