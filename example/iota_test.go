package example

import (
	"math"
	"testing"
)

func TestCalculateArea(t *testing.T) {
	tests := []struct {
		name   string
		shape  Shape
		expect float64
	}{
		{
			name: "circle with radius 1",
			shape: Shape{
				Type:   CircleType,
				Circle: &Circle{Radius: 1.0},
			},
			expect: math.Pi,
		},
		{
			name: "circle with radius 0",
			shape: Shape{
				Type:   CircleType,
				Circle: &Circle{Radius: 0.0},
			},
			expect: 0.0,
		},
		{
			name: "rectangle with positive dimensions",
			shape: Shape{
				Type: RectangleType,
				Rect: &Rectangle{Width: 5.0, Height: 2.0},
			},
			expect: 10.0,
		},
		{
			name: "rectangle with zero dimensions",
			shape: Shape{
				Type: RectangleType,
				Rect: &Rectangle{Width: 0.0, Height: 0.0},
			},
			expect: 0.0,
		},
		//{
		//	name: "triangle with positive dimensions",
		//	shape: Shape{
		//		Type:     TriangleType,
		//		Triangle: &Triangle{Base: 6.0, Height: 4.0},
		//	},
		//	expect: 12.0,
		//},
		//{
		//	name: "triangle with zero base",
		//	shape: Shape{
		//		Type:     TriangleType,
		//		Triangle: &Triangle{Base: 0.0, Height: 5.0},
		//	},
		//	expect: 0.0,
		//},
		{
			name: "circle with nil pointer",
			shape: Shape{
				Type:   CircleType,
				Circle: nil,
			},
			expect: 0.0,
		},
		{
			name: "rectangle with nil pointer",
			shape: Shape{
				Type: RectangleType,
				Rect: nil,
			},
			expect: 0.0,
		},
		{
			name: "triangle with nil pointer",
			shape: Shape{
				Type:     TriangleType,
				Triangle: nil,
			},
			expect: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateArea(tt.shape)
			if got != tt.expect {
				t.Errorf("CalculateArea() = %v, want %v", got, tt.expect)
			}
		})
	}
}
