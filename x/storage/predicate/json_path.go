package predicate

import (
	"fmt"
	"strings"

	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
)

// PathStep is one step into a decoded JSON document. Exactly one of the
// three selectors is set.
type PathStep struct {
	Field *string
	Index *int
	Any   bool
}

func MkPathField(name string) PathStep {
	return PathStep{Field: &name}
}

func MkPathIndex(i int) PathStep {
	return PathStep{Index: &i}
}

func MkPathAny() PathStep {
	return PathStep{Any: true}
}

// JSONPath addresses values inside the JSON encoding of a record,
// as produced by shared.JSONMarshal and decoded with encoding/json into any.
type JSONPath []PathStep

func (p JSONPath) String() string {
	var b strings.Builder
	for i, s := range p {
		switch {
		case s.Field != nil:
			if i > 0 {
				b.WriteString(".")
			}
			b.WriteString(*s.Field)
		case s.Index != nil:
			fmt.Fprintf(&b, "[%d]", *s.Index)
		case s.Any:
			b.WriteString("[*]")
		}
	}
	return b.String()
}

// TypeDiscriminatorField is the union discriminator field name used by the
// mkunion JSON encoding.
const TypeDiscriminatorField = "$type"

// pathAppend copies before appending; resolution branches over union
// variants share prefixes and must not alias the same backing array.
func pathAppend(prefix JSONPath, step PathStep) JSONPath {
	result := make(JSONPath, len(prefix), len(prefix)+1)
	copy(result, prefix)
	return append(result, step)
}

// ResolveJSONPaths translates a user-facing location such as
//
//	Data.Age
//	Data["testutil.Branch"].Name
//	Data.Name
//	Data["$type"]
//
// into one or more concrete paths inside the mkunion JSON encoding of a
// value described by shape s. More than one path is returned when the
// location crosses a union without naming the variant; the paths are
// alternatives (logical OR). A location that names a field or variant the
// shape does not have is an error, not an empty result.
func ResolveJSONPaths(s shape.Shape, location string) ([]JSONPath, error) {
	locs, err := schema.ParseLocation(location)
	if err != nil {
		return nil, fmt.Errorf("predicate.ResolveJSONPaths: parse %q: %w", location, err)
	}

	result, err := resolveJSONPaths(s, locs, nil)
	if err != nil {
		return nil, fmt.Errorf("predicate.ResolveJSONPaths: location %q: %w", location, err)
	}
	return result, nil
}

func resolveJSONPaths(s shape.Shape, locs []schema.Location, prefix JSONPath) ([]JSONPath, error) {
	if len(locs) == 0 {
		return []JSONPath{prefix}, nil
	}

	switch x := s.(type) {
	case *shape.RefName:
		ref, found := shape.LookupShape(x)
		if !found {
			// Unregistered references cannot be validated; resolve the
			// remaining location verbatim, like shape.Any.
			return resolveJSONPathsAny(locs, prefix), nil
		}
		return resolveJSONPaths(shape.IndexWith(ref, x), locs, prefix)

	case *shape.PointerLike:
		return resolveJSONPaths(x.Type, locs, prefix)

	case *shape.AliasLike:
		return resolveJSONPaths(x.Type, locs, prefix)

	case *shape.Any:
		return resolveJSONPathsAny(locs, prefix), nil

	case *shape.PrimitiveLike:
		return nil, fmt.Errorf("cannot descend %s into primitive value", schema.LocationToStr(locs))

	case *shape.ListLike:
		switch loc := locs[0].(type) {
		case *schema.LocationIndex:
			return resolveJSONPaths(x.Element, locs[1:], pathAppend(prefix, MkPathIndex(loc.Index)))
		case *schema.LocationAnything:
			return resolveJSONPaths(x.Element, locs[1:], pathAppend(prefix, MkPathAny()))
		default:
			return nil, fmt.Errorf("list requires [index] or [*], got %s", schema.LocationToStr(locs[:1]))
		}

	case *shape.MapLike:
		switch loc := locs[0].(type) {
		case *schema.LocationField:
			return resolveJSONPaths(x.Val, locs[1:], pathAppend(prefix, MkPathField(loc.Name)))
		case *schema.LocationAnything:
			return resolveJSONPaths(x.Val, locs[1:], pathAppend(prefix, MkPathAny()))
		default:
			return nil, fmt.Errorf("map requires [\"key\"] or [*], got %s", schema.LocationToStr(locs[:1]))
		}

	case *shape.StructLike:
		loc, ok := locs[0].(*schema.LocationField)
		if !ok {
			return nil, fmt.Errorf("struct %s.%s requires a field name, got %s", x.PkgName, x.Name, schema.LocationToStr(locs[:1]))
		}
		for _, field := range x.Fields {
			jsonName := jsonFieldName(field)
			if jsonName == "" {
				continue
			}
			if field.Name == loc.Name || jsonName == loc.Name {
				return resolveJSONPaths(field.Type, locs[1:], pathAppend(prefix, MkPathField(jsonName)))
			}
		}
		return nil, fmt.Errorf("struct %s.%s has no field %q", x.PkgName, x.Name, loc.Name)

	case *shape.UnionLike:
		return resolveJSONPathsUnion(x, locs, prefix)
	}

	return nil, fmt.Errorf("unsupported shape %T", s)
}

func resolveJSONPathsUnion(x *shape.UnionLike, locs []schema.Location, prefix JSONPath) ([]JSONPath, error) {
	// [*] selects every variant.
	if _, ok := locs[0].(*schema.LocationAnything); ok {
		return resolveOverVariants(x, x.Variant, locs[1:], prefix)
	}

	loc, ok := locs[0].(*schema.LocationField)
	if !ok {
		return nil, fmt.Errorf("union %s.%s requires a field, variant or [*], got %s", x.PkgName, x.Name, schema.LocationToStr(locs[:1]))
	}

	// The discriminator is queryable directly: Data["$type"] = :t
	if loc.Name == TypeDiscriminatorField {
		if len(locs) > 1 {
			return nil, fmt.Errorf("%s is terminal, cannot descend %s", TypeDiscriminatorField, schema.LocationToStr(locs[1:]))
		}
		return []JSONPath{pathAppend(prefix, MkPathField(TypeDiscriminatorField))}, nil
	}

	// Explicit variant: Data["testutil.Branch"].Name or Data["Branch"].Name
	for _, variant := range x.Variant {
		key := JSONVariantName(variant)
		if loc.Name == key || loc.Name == variantBareName(variant) {
			return resolveJSONPaths(variant, locs[1:], pathAppend(prefix, MkPathField(key)))
		}
	}

	// Bare field: expand over every variant that has it.
	var result []JSONPath
	var variantNames []string
	for _, variant := range x.Variant {
		variantNames = append(variantNames, JSONVariantName(variant))
		paths, err := resolveJSONPaths(variant, locs, pathAppend(prefix, MkPathField(JSONVariantName(variant))))
		if err != nil {
			continue
		}
		result = append(result, paths...)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("union %s.%s: no variant (%s) has %s", x.PkgName, x.Name, strings.Join(variantNames, ", "), schema.LocationToStr(locs))
	}
	return result, nil
}

func resolveOverVariants(x *shape.UnionLike, variants []shape.Shape, locs []schema.Location, prefix JSONPath) ([]JSONPath, error) {
	var result []JSONPath
	for _, variant := range variants {
		paths, err := resolveJSONPaths(variant, locs, pathAppend(prefix, MkPathField(JSONVariantName(variant))))
		if err != nil {
			continue
		}
		result = append(result, paths...)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("union %s.%s: no variant matches %s", x.PkgName, x.Name, schema.LocationToStr(locs))
	}
	return result, nil
}

// resolveJSONPathsAny maps the remaining location verbatim when the shape
// carries no information to validate against.
func resolveJSONPathsAny(locs []schema.Location, prefix JSONPath) []JSONPath {
	path := prefix
	for _, loc := range locs {
		switch l := loc.(type) {
		case *schema.LocationField:
			path = append(path, MkPathField(l.Name))
		case *schema.LocationIndex:
			path = append(path, MkPathIndex(l.Index))
		case *schema.LocationAnything:
			path = append(path, MkPathAny())
		}
	}
	return []JSONPath{path}
}

// JSONVariantName returns the union variant key used by the mkunion JSON
// encoding, e.g. "testutil.Branch". It mirrors generators.JSONVariantName.
func JSONVariantName(s shape.Shape) string {
	switch x := s.(type) {
	case *shape.RefName:
		return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
	case *shape.AliasLike:
		return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
	case *shape.StructLike:
		return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
	case *shape.UnionLike:
		return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
	case *shape.PointerLike:
		return JSONVariantName(x.Type)
	}
	return ""
}

func variantBareName(s shape.Shape) string {
	switch x := s.(type) {
	case *shape.RefName:
		return x.Name
	case *shape.AliasLike:
		return x.Name
	case *shape.StructLike:
		return x.Name
	case *shape.UnionLike:
		return x.Name
	case *shape.PointerLike:
		return variantBareName(x.Type)
	}
	return ""
}

func jsonFieldName(field *shape.FieldLike) string {
	name := shape.TagGetValue(field.Tags, "json", field.Name)
	if idx := strings.Index(name, ","); idx >= 0 {
		name = name[:idx]
		if name == "" {
			name = field.Name
		}
	}
	if name == "-" {
		return ""
	}
	return name
}
