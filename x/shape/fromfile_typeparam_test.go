package shape

import (
	"testing"
)

func TestParseUnionNameWithTypeParams(t *testing.T) {
	tests := []struct {
		input              string
		expectedName       string
		expectedTypeParams []string
	}{
		{
			input:              "Tree",
			expectedName:       "Tree",
			expectedTypeParams: nil,
		},
		{
			input:              "Tree[T]",
			expectedName:       "Tree",
			expectedTypeParams: []string{"T"},
		},
		{
			input:              "Result[T, E]",
			expectedName:       "Result",
			expectedTypeParams: []string{"T", "E"},
		},
		{
			input:              "Complex[A, B, C]",
			expectedName:       "Complex",
			expectedTypeParams: []string{"A", "B", "C"},
		},
		{
			input:              "SpacedParams[A , B , C]",
			expectedName:       "SpacedParams",
			expectedTypeParams: []string{"A", "B", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, params := parseUnionNameWithTypeParams(tt.input)

			if name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, name)
			}

			if len(params) != len(tt.expectedTypeParams) {
				t.Errorf("Expected %d type params, got %d", len(tt.expectedTypeParams), len(params))
				return
			}

			for i, param := range params {
				if param != tt.expectedTypeParams[i] {
					t.Errorf("Expected param[%d] = %q, got %q", i, tt.expectedTypeParams[i], param)
				}
			}
		})
	}
}

func TestInferFromFileWithTypeParams(t *testing.T) {
	// Test that unions with explicit type params are correctly parsed
	unionName, typeParams := parseUnionNameWithTypeParams("Option[T]")
	if unionName != "Option" {
		t.Errorf("Expected union name 'Option', got %q", unionName)
	}
	if len(typeParams) != 1 || typeParams[0] != "T" {
		t.Errorf("Expected type params [T], got %v", typeParams)
	}

	unionName2, typeParams2 := parseUnionNameWithTypeParams("Result[V, E]")
	if unionName2 != "Result" {
		t.Errorf("Expected union name 'Result', got %q", unionName2)
	}
	if len(typeParams2) != 2 || typeParams2[0] != "V" || typeParams2[1] != "E" {
		t.Errorf("Expected type params [V, E], got %v", typeParams2)
	}
}

func TestFormatTypeParamsForTag(t *testing.T) {
	tests := []struct {
		name     string
		params   []TypeParam
		expected string
	}{
		{
			name:     "empty params",
			params:   []TypeParam{},
			expected: "",
		},
		{
			name:     "single param",
			params:   []TypeParam{{Name: "T"}},
			expected: "[T]",
		},
		{
			name:     "multiple params",
			params:   []TypeParam{{Name: "K"}, {Name: "V"}},
			expected: "[K, V]",
		},
		{
			name:     "three params",
			params:   []TypeParam{{Name: "A"}, {Name: "B"}, {Name: "C"}},
			expected: "[A, B, C]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTypeParamsForTag(tt.params)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
