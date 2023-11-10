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
