package schema

import "strings"

var _ RuleMatcher = (*WhenField[any])(nil)

type WhenField[A any] struct {
	t          A
	path       []string
	typeMapDef TypeMapDefinition
}

func (r *WhenField[A]) SchemaToUnionType(x any, schema Schema, config *goConfig) (Schema, bool) {
	return nil, false
}

func (r *WhenField[A]) MapDefFor(x *Map, path []string, config *goConfig) (TypeMapDefinition, bool) {
	if len(r.path) == 1 && r.path[0] == "*" {
		return r.typeMapDef, true
	}

	if len(r.path) > 1 && r.path[0] == "*" {
		if len(path) < len(r.path)-1 {
			return nil, false
		}

		isAnyPath := r.path[0] == "*"
		if isAnyPath {
			pathLen := len(r.path)
			for i := 1; i < pathLen; i++ {
				//parts := strings.Split(r.path[1], "?.")
				// compare from the end
				if r.path[len(r.path)-i] != path[len(path)-i] && r.path[len(r.path)-i] != "*" {
					return nil, false
				}
			}
			return r.typeMapDef, true
		}
	}

	if len(path) != len(r.path) {
		return nil, false
	}

	for i := range r.path {
		parts := strings.Split(r.path[i], "?.")
		if path[i] != parts[0] && parts[0] != "*" {
			return nil, false
		}

		if len(parts) != 2 {
			continue
		}

		found := false
		for _, f := range x.Field {
			if f.Name == parts[1] {
				found = true
				break
			}
		}

		if !found {
			return nil, false
		}
	}

	return r.typeMapDef, true
}
