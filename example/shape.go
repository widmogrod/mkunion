package example

import (
	"fmt"
	"math"
)

// Shape represents different geometric shapes
// This file demonstrates mkunion's union type capabilities

// --8<-- [start:shape-def]

//go:tag mkunion:"Shape"
type (
	Circle struct {
		Radius float64
	}
	Rectangle struct {
		Width  float64
		Height float64
	}
	Square struct {
		Side float64
	}
)

// --8<-- [end:shape-def]

// --8<-- [start:calculate-area]

func CalculateArea(s Shape) float64 {
	return MatchShapeR1(
		s,
		func(x *Circle) float64 {
			return math.Pi * x.Radius * x.Radius
		},
		func(x *Rectangle) float64 {
			return x.Width * x.Height
		},
		func(x *Square) float64 {
			return x.Side * x.Side
		},
	)
}

// --8<-- [end:calculate-area]

// --8<-- [start:calculate-perimeter]

func CalculatePerimeter(s Shape) float64 {
	return MatchShapeR1(
		s,
		func(x *Circle) float64 {
			return 2 * math.Pi * x.Radius
		},
		func(x *Rectangle) float64 {
			return 2 * (x.Width + x.Height)
		},
		func(x *Square) float64 {
			return 4 * x.Side
		},
	)
}

// --8<-- [end:calculate-perimeter]

// --8<-- [start:describe-shape]

func DescribeShape(s Shape) string {
	return MatchShapeR1(
		s,
		func(x *Circle) string {
			return fmt.Sprintf("Circle with radius %.2f", x.Radius)
		},
		func(x *Rectangle) string {
			return fmt.Sprintf("Rectangle with width %.2f and height %.2f", x.Width, x.Height)
		},
		func(x *Square) string {
			return fmt.Sprintf("Square with side %.2f", x.Side)
		},
	)
}

// --8<-- [end:describe-shape]

// --8<-- [start:match-def]

//go:tag mkmatch
type MatchShapes[A, B Shape] interface {
	MatchCircles(x, y *Circle)
	MatchRectangleAny(x *Rectangle, y any)
	Finally(x, y any)
}

// --8<-- [end:match-def]

// --8<-- [start:match-shapes]

func CompareShapes(x, y Shape) string {
	return MatchShapesR1(
		x, y,
		func(x0 *Circle, x1 *Circle) string {
			return fmt.Sprintf("Two circles with radii %.2f and %.2f", x0.Radius, x1.Radius)
		},
		func(x0 *Rectangle, x1 any) string {
			return fmt.Sprintf("Rectangle (%.2fx%.2f) meets %T", x0.Width, x0.Height, x1)
		},
		func(x0 any, x1 any) string {
			return fmt.Sprintf("Finally: %T meets %T", x0, x1)
		},
	)
}

// --8<-- [end:match-shapes]
