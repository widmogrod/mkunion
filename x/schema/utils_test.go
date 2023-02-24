package schema

import "testing"

func TestCompare(t *testing.T) {
	useCases := map[string]struct {
		a, b Schema
		cmp  int
	}{
		"none and none = 0": {
			a:   &None{},
			b:   &None{},
			cmp: 0,
		},
		"none and true = -1": {
			a:   &None{},
			b:   MkBool(true),
			cmp: -1,
		},
		"none and false = -1": {
			a:   &None{},
			b:   MkBool(false),
			cmp: -1,
		},
		"none and number = -1": {
			a:   &None{},
			b:   MkInt(1),
			cmp: -1,
		},
		"none and string = -1": {
			a:   &None{},
			b:   MkString("some cool string"),
			cmp: -1,
		},
		"none and list = -1": {
			a:   &None{},
			b:   &List{},
			cmp: -1,
		},
		"none and map = -1": {
			a:   &None{},
			b:   &Map{},
			cmp: -1,
		},
		"true and none = 1": {
			a:   MkBool(true),
			b:   &None{},
			cmp: 1,
		},
		"true and true = 0": {
			a:   MkBool(true),
			b:   MkBool(true),
			cmp: 0,
		},
		"true and false = 1": {
			a:   MkBool(true),
			b:   MkBool(false),
			cmp: 1,
		},
		"true and number = -1": {
			a:   MkBool(true),
			b:   MkInt(1),
			cmp: -1,
		},
		"true and string = -1": {
			a:   MkBool(true),
			b:   MkString("some cool string"),
			cmp: -1,
		},
		"true and list = -1": {
			a:   MkBool(true),
			b:   &List{},
			cmp: -1,
		},
		"true and map = -1": {
			a:   MkBool(true),
			b:   &Map{},
			cmp: -1,
		},
		"string and none = 1": {
			a:   MkString("some cool string"),
			b:   &None{},
			cmp: 1,
		},
		"string and true = 1": {
			a:   MkString("some cool string"),
			b:   MkBool(true),
			cmp: 1,
		},
		"string and false = 1": {
			a:   MkString("some cool string"),
			b:   MkBool(false),
			cmp: 1,
		},
		"string and number = -1": {
			a:   MkString("some cool string"),
			b:   MkInt(1),
			cmp: 1,
		},
		"string and string = 0": {
			a:   MkString("some cool string"),
			b:   MkString("some cool string"),
			cmp: 0,
		},
		"string and list = -1": {
			a:   MkString("some cool string"),
			b:   &List{},
			cmp: -1,
		},
		"string and map = -1": {
			a:   MkString("some cool string"),
			b:   &Map{},
			cmp: -1,
		},
		"list and none = 1": {
			a:   &List{},
			b:   &None{},
			cmp: 1,
		},
		"list and true = 1": {
			a:   &List{},
			b:   MkBool(true),
			cmp: 1,
		},
		"list and false = 1": {
			a:   &List{},
			b:   MkBool(false),
			cmp: 1,
		},
		"list and number = 1": {
			a:   &List{},
			b:   MkInt(1),
			cmp: 1,
		},
		"list and string = 1": {
			a:   &List{},
			b:   MkString("some cool string"),
			cmp: 1,
		},
		"list and list = 0": {
			a: MkList(MkInt(1), MkInt(2), MkInt(3)),
			b: MkList(MkInt(1), MkInt(2), MkInt(3)),
		},
		"list and map = -1": {
			a:   &List{},
			b:   &Map{},
			cmp: -1,
		},
		"map and none = 1": {
			a:   &Map{},
			b:   &None{},
			cmp: 1,
		},
		"map and true = 1": {
			a:   &Map{},
			b:   MkBool(true),
			cmp: 1,
		},
		"map and false = 1": {
			a:   &Map{},
			b:   MkBool(false),
			cmp: 1,
		},
		"map and number = 1": {
			a:   &Map{},
			b:   MkInt(1),
			cmp: 1,
		},
		"map and string = 1": {
			a:   &Map{},
			b:   MkString("some cool string"),
			cmp: 1,
		},
		"map and list = 1": {
			a:   &Map{},
			b:   &List{},
			cmp: 1,
		},
		"map and map = 0": {
			a:   MkMap(MkField("a", MkInt(1)), MkField("b", MkInt(2)), MkField("c", MkInt(3))),
			b:   MkMap(MkField("a", MkInt(1)), MkField("b", MkInt(2)), MkField("c", MkInt(3))),
			cmp: 0,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			cmp := Compare(uc.a, uc.b)
			if cmp != uc.cmp {
				t.Fatalf("expected %d, got %d", uc.cmp, cmp)
			}
		})
	}
}
