package example

import (
	"fmt"
	"testing"
)

func ExampleCalculateSpeed() {
	distance := NewDistance(100.0) // meters
	time := NewTime(10.0)          // seconds
	if positive, ok := time.(*PositiveTime[Seconds]); ok {
		speed := CalculateSpeed(distance, positive)
		fmt.Printf("Speed: %.2f m/s\n", speed.value)
	}
}

func TestCalculateSpeed(t *testing.T) {
	tests := []struct {
		name     string
		distance *Distance[Meters]
		time     Time[Seconds]
		want     float64
	}{
		{
			name:     "valid positive distance and time",
			distance: NewDistance(100.0),
			time:     NewTime(10.0),
			want:     10.0,
		},
		{
			name:     "very small non-zero values",
			distance: NewDistance(0.001),
			time:     NewTime(0.0001),
			want:     10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSpeed(tt.distance, tt.time.(*PositiveTime[Seconds]))

			if got.value != tt.want {
				t.Errorf("CalculateSpeed() = %v, want %v", got.value, tt.want)
			}
		})
	}
}
