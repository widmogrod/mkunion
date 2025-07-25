package example

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/widmogrod/mkunion/x/shared"
)

// --8<-- [start:json]

func TestShapeJSON(t *testing.T) {
	shape := &Rectangle{
		Width:  10,
		Height: 20,
	}
	result, err := shared.JSONMarshal[Shape](shape)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(result))
	// Output: {"$type":"example.Rectangle","example.Rectangle":{"Width":10,"Height":20}}

	shape2, err := shared.JSONUnmarshal[Shape](result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%#v", shape2)
	// Output: &example.Rectangle{Width:10, Height:20}
}

// --8<-- [end:json]

func TestCalculateArea(t *testing.T) {
	tests := []struct {
		name     string
		shape    Shape
		expected float64
	}{
		{
			name:     "Circle area",
			shape:    &Circle{Radius: 5},
			expected: 78.53981633974483, // π * 5^2
		},
		{
			name:     "Rectangle area",
			shape:    &Rectangle{Width: 10, Height: 20},
			expected: 200,
		},
		{
			name:     "Square area",
			shape:    &Square{Side: 10},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CalculateArea(tt.shape)
			if actual != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, actual)
			}
		})
	}
}

func TestDescribeShape(t *testing.T) {
	tests := []struct {
		name     string
		shape    Shape
		expected string
	}{
		{
			name:     "Describe Circle",
			shape:    &Circle{Radius: 5},
			expected: "Circle with radius 5.00",
		},
		{
			name:     "Describe Rectangle",
			shape:    &Rectangle{Width: 10, Height: 20},
			expected: "Rectangle with width 10.00 and height 20.00",
		},
		{
			name:     "Describe Square",
			shape:    &Square{Side: 10},
			expected: "Square with side 10.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := DescribeShape(tt.shape)
			if actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestCompareShapes(t *testing.T) {
	tests := []struct {
		name     string
		shape1   Shape
		shape2   Shape
		expected string
	}{
		{
			name:     "Two circles",
			shape1:   &Circle{Radius: 5},
			shape2:   &Circle{Radius: 10},
			expected: "Two circles with radii 5.00 and 10.00",
		},
		{
			name:     "Rectangle and Circle",
			shape1:   &Rectangle{Width: 10, Height: 20},
			shape2:   &Circle{Radius: 5},
			expected: "Rectangle (10.00x20.00) meets *example.Circle",
		},
		{
			name:     "Square and Rectangle",
			shape1:   &Square{Side: 10},
			shape2:   &Rectangle{Width: 10, Height: 20},
			expected: "Finally: *example.Square meets *example.Rectangle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CompareShapes(tt.shape1, tt.shape2)
			if actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func ExampleShapeToJSON() {
	var shape Shape = &Circle{
		Radius: 10,
	}
	result, _ := shared.JSONMarshal(shape)
	fmt.Println(string(result))
	// Output: {"$type":"example.Circle","example.Circle":{"Radius":10}}
}

func ExampleShapeFromJSON() {
	input := []byte(`{"$type":"example.Circle","example.Circle":{"Radius":10}}`)
	shape, _ := shared.JSONUnmarshal[Shape](input)
	fmt.Printf("%#v", shape)
	// Output: &example.Circle{Radius:10}
}

func TestShapeJSONRoundTrip(t *testing.T) {
	shapes := []Shape{
		&Circle{Radius: 5},
		&Rectangle{Width: 10, Height: 20},
		&Square{Side: 10},
	}

	for _, shape := range shapes {
		t.Run(fmt.Sprintf("%T", shape), func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := shared.JSONMarshal(shape)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Verify JSON structure
			var rawJSON map[string]json.RawMessage
			if err := json.Unmarshal(jsonData, &rawJSON); err != nil {
				t.Fatalf("failed to unmarshal raw JSON: %v", err)
			}

			// Check for $type field
			if _, ok := rawJSON["$type"]; !ok {
				t.Error("missing $type field in JSON")
			}

			// Unmarshal back
			unmarshaled, err := shared.JSONUnmarshal[Shape](jsonData)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Compare
			if fmt.Sprintf("%+v", shape) != fmt.Sprintf("%+v", unmarshaled) {
				t.Errorf("roundtrip failed: original %+v, got %+v", shape, unmarshaled)
			}
		})
	}
}
