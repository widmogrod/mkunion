package example

import (
	"math"
	"testing"
)

func TestCalculateIotaArea(t *testing.T) {
	tests := []struct {
		name   string
		shape  IotaShape
		expect float64
	}{
		{
			name: "circle with radius 1",
			shape: IotaShape{
				Type:   IotaCircleType,
				Circle: &IotaCircle{Radius: 1.0},
			},
			expect: math.Pi,
		},
		{
			name: "circle with radius 0",
			shape: IotaShape{
				Type:   IotaCircleType,
				Circle: &IotaCircle{Radius: 0.0},
			},
			expect: 0.0,
		},
		{
			name: "rectangle with positive dimensions",
			shape: IotaShape{
				Type: IotaRectangleType,
				Rect: &IotaRectangle{Width: 5.0, Height: 2.0},
			},
			expect: 10.0,
		},
		{
			name: "rectangle with zero dimensions",
			shape: IotaShape{
				Type: IotaRectangleType,
				Rect: &IotaRectangle{Width: 0.0, Height: 0.0},
			},
			expect: 0.0,
		},
		//{
		//	name: "triangle with positive dimensions",
		//	shape: IotaShape{
		//		Type:     IotaTriangleType,
		//		Triangle: &IotaTriangle{Base: 6.0, Height: 4.0},
		//	},
		//	expect: 12.0,
		//},
		//{
		//	name: "triangle with zero base",
		//	shape: IotaShape{
		//		Type:     IotaTriangleType,
		//		Triangle: &IotaTriangle{Base: 0.0, Height: 5.0},
		//	},
		//	expect: 0.0,
		//},
		{
			name: "circle with nil pointer",
			shape: IotaShape{
				Type:   IotaCircleType,
				Circle: nil,
			},
			expect: 0.0,
		},
		{
			name: "rectangle with nil pointer",
			shape: IotaShape{
				Type: IotaRectangleType,
				Rect: nil,
			},
			expect: 0.0,
		},
		{
			name: "triangle with nil pointer",
			shape: IotaShape{
				Type:     IotaTriangleType,
				Triangle: nil,
			},
			expect: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateIotaArea(tt.shape)
			if got != tt.expect {
				t.Errorf("CalculateIotaArea() = %v, want %v", got, tt.expect)
			}
		})
	}
}
