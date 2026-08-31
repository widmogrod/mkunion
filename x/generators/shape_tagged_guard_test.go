package generators

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
)

func TestGuardToString(t *testing.T) {
	useCases := map[string]struct {
		in   shape.Guard
		want string
	}{
		"nil guard": {nil, "nil"},
		"required":  {&shape.Required{}, "&shape.Required{}"},
		"empty enum": {
			&shape.Enum{},
			"&shape.Enum{\n}",
		},
		"enum with values": {
			&shape.Enum{Val: []string{"a", "b"}},
			"&shape.Enum{\n\tVal: []string{\n\t\t\"a\",\n\t\t\"b\",\n\t},\n}",
		},
		"empty and guard": {
			&shape.AndGuard{},
			"&shape.AndGuard{\n}",
		},
		"and guard nests recursively": {
			&shape.AndGuard{L: []shape.Guard{
				&shape.Required{},
				&shape.Enum{Val: []string{"x"}},
			}},
			"&shape.AndGuard{\n" +
				"\tGuards: []shape.Guard{\n" +
				"\t\t&shape.Required{},\n" +
				"\t\t&shape.Enum{\n" +
				"\t\t\tVal: []string{\n" +
				"\t\t\t\t\"x\",\n" +
				"\t\t\t},\n" +
				"\t\t},\n" +
				"\t},\n" +
				"}",
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.want, GuardToString(uc.in))
		})
	}
}

func TestPtrToString(t *testing.T) {
	assert.Equal(t, "nil", PtrToString(nil))
	s := "x"
	assert.Equal(t, `shape.Ptr("x")`, PtrToString(&s))
}
