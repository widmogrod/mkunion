package shape

const (
	returnsPkgImportName = "github.com/widmogrod/mkunion/f"
	returnsName          = "Returns"
)

// ReturnsOf reports whether a field is an embedded f.Returns[R] marker and,
// when it is, returns R. Such a field is a phantom: it declares the answer
// type of a union variant and carries no data, so serde and exporters skip it.
func ReturnsOf(field *FieldLike) (Shape, bool) {
	ref, ok := field.Type.(*RefName)
	if !ok || ref.Name != returnsName || ref.PkgImportName != returnsPkgImportName {
		return nil, false
	}
	if len(ref.Indexed) != 1 {
		return nil, false
	}
	return ref.Indexed[0], true
}

// IsPhantomField reports whether a struct field carries no data and must be
// skipped when the struct is serialised or exported.
func IsPhantomField(field *FieldLike) bool {
	_, ok := ReturnsOf(field)
	return ok
}

// UnionDeclaresReturns reports whether at least one variant of the union
// embeds f.Returns[R]. Such a union gets a typed handler generated for it.
func UnionDeclaresReturns(union *UnionLike) bool {
	for _, v := range union.Variant {
		st, ok := v.(*StructLike)
		if !ok {
			continue
		}
		for _, field := range st.Fields {
			if IsPhantomField(field) {
				return true
			}
		}
	}
	return false
}
