package predicate

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOptimizePredicates(t *testing.T) {
	t.Run("optimize double negation", func(t *testing.T) {
		p := &Not{
			P: &Not{
				P: &Compare{
					Location:  "x",
					Operation: "=",
					BindValue: &BindValue{BindName: ":x"},
				},
			},
		}
		expected := &Compare{
			Location:  "x",
			Operation: "=",
			BindValue: &BindValue{BindName: ":x"},
		}
		assert.Equal(t, expected, Optimize(p))
	})

	t.Run("optimize one element AND", func(t *testing.T) {
		p := &And{
			L: []Predicate{
				&Compare{
					Location:  "x",
					Operation: "=",
					BindValue: &BindValue{BindName: ":x"},
				},
			},
		}
		expected := &Compare{
			Location:  "x",
			Operation: "=",
			BindValue: &BindValue{BindName: ":x"},
		}
		assert.Equal(t, expected, Optimize(p))
	})

	t.Run("optimize recurses into children", func(t *testing.T) {
		cmp1 := &Compare{
			Location:  "x",
			Operation: "=",
			BindValue: &BindValue{BindName: ":x"},
		}
		cmp2 := &Compare{
			Location:  "y",
			Operation: "=",
			BindValue: &BindValue{BindName: ":y"},
		}
		p := &And{
			L: []Predicate{
				&Not{P: &Not{P: cmp1}},
				&Or{L: []Predicate{cmp2}},
			},
		}
		expected := &And{
			L: []Predicate{cmp1, cmp2},
		}
		assert.Equal(t, expected, Optimize(p))
	})

	t.Run("optimize output of Parse", func(t *testing.T) {
		// Parse always wraps in Or{And{...}}, Optimize must unwrap
		// all the way down to the single comparison.
		p, err := Parse("Age <= 20")
		assert.NoError(t, err)

		optimized := Optimize(p)
		assert.IsType(t, &Compare{}, optimized)
	})

	t.Run("optimize double negation below NOT", func(t *testing.T) {
		cmp := &Compare{
			Location:  "x",
			Operation: "=",
			BindValue: &BindValue{BindName: ":x"},
		}
		p := &Not{P: &And{L: []Predicate{&Not{P: &Not{P: cmp}}}}}
		expected := &Not{P: cmp}
		assert.Equal(t, expected, Optimize(p))
	})

	t.Run("optimize one element OR", func(t *testing.T) {
		p := &Or{
			L: []Predicate{
				&Compare{
					Location:  "x",
					Operation: "=",
					BindValue: &BindValue{BindName: ":x"},
				},
			},
		}
		expected := &Compare{
			Location:  "x",
			Operation: "=",
			BindValue: &BindValue{BindName: ":x"},
		}
		assert.Equal(t, expected, Optimize(p))
	})
}
