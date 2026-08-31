package predicate

import (
	"testing"

	"github.com/widmogrod/mkunion/x/schema"
)

func TestParseRejectsInvalidOperators(t *testing.T) {
	useCases := []string{
		"Age =< 20",
		"Age === 20",
		"Age =! 20",
		"Age <=> 20",
	}
	for _, uc := range useCases {
		t.Run(uc, func(t *testing.T) {
			p, err := Parse(uc)
			if err == nil {
				t.Fatalf("expected parse error for invalid operator, got predicate %#v", p)
			}
		})
	}
}

func TestParseFieldsWithKeywordPrefix(t *testing.T) {
	useCases := []string{
		`ORDER = 1`,
		`ANDrzej = "abc"`,
		`NOTE = :param`,
		`ID = 1 AND ORDER = 2`,
	}
	for _, uc := range useCases {
		t.Run(uc, func(t *testing.T) {
			_, err := Parse(uc)
			if err != nil {
				t.Fatalf("expected field name starting with keyword to parse, got error: %v", err)
			}
		})
	}
}

func TestParseParentheses(t *testing.T) {
	data := schema.FromGo(map[string]any{
		"ID":      "123",
		"Age":     20,
		"Visible": true,
	})

	useCases := []struct {
		value  string
		result bool
	}{
		// Parenthesized: (true OR false) AND false = false.
		// Without grouping support this would parse as
		// true OR (false AND false) = true.
		{
			value:  `(ID = "123" OR Age = 999) AND Visible = false`,
			result: false,
		},
		{
			value:  `(ID = "999" OR Age = 20) AND Visible = true`,
			result: true,
		},
		{
			value:  `NOT (ID = "123" OR Age = 999)`,
			result: false,
		},
		{
			value:  `NOT (ID = "999" AND Age = 999) AND Visible = true`,
			result: true,
		},
	}
	for _, uc := range useCases {
		t.Run(uc.value, func(t *testing.T) {
			p, err := Parse(uc.value)
			if err != nil {
				t.Fatal(err)
			}

			if result := EvaluateSchema(p, data, nil); result != uc.result {
				t.Fatalf("expected %v, got %v", uc.result, result)
			}
		})
	}
}
