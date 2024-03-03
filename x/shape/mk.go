package shape

import (
	"reflect"
	"strings"
)

func MkRefNameFromReflect(x reflect.Type) *RefName {
	return MkRefNameFromString(ToGoFullTypeNameFromReflect(x))
}

func MkRefNameFromString(x string) *RefName {
	x = strings.TrimSpace(x)
	// parse type name, extract package name and type name, and if type is indexed
	// then extract all indexed types

	// parse string
	// if first char is "*" then it's a pointer
	// type name is until char is "[" then it's a slice
	// if type name is until char is "." then it's a package name
	// if type name is until char is "]" then it's a indexed type
	// if type name is until char is "," then it's a next indexed type

	result := &RefName{}

	// example: "*some.Type2[int, some.Type[int]]"
	if strings.HasPrefix(x, "*") {
		x = x[1:]
	}

	name := x
	// example "some.Type2[int, some.Type[int]]"
	// example "some.Type2[some.Type[int, some.Type[float64]]"
	if index := strings.Index(x, "["); index != -1 {
		// example: some.Type2
		name = x[:index]

		//example: int, some.Type[int]
		rest := x[index:]
		// remove first "[" and last "]"
		rest = rest[1 : len(rest)-1]

		// scan for "," and split
		commaIndex := strings.Index(rest, ",")

		// scan for "["
		leftBracketIndex := strings.Index(rest, "[")

		// example: "int, some.Type[int]"
		hasComma := commaIndex != -1

		// example: "int, some.Type[int]"
		// 	hasTypeParam == true
		// example: "some.Type[int, some.Type[float64]]"
		//	hasTypeParam == false
		hasTypeParam := leftBracketIndex != -1
		isTypeParamFirst := commaIndex > leftBracketIndex

		if hasComma && !(hasTypeParam && isTypeParamFirst) {
			first := rest[:commaIndex]
			rest := rest[commaIndex+1:]

			result.Indexed = append(result.Indexed, MkRefNameFromString(first))
			result.Indexed = append(result.Indexed, MkRefNameFromString(rest))
		} else {
			result.Indexed = append(result.Indexed, MkRefNameFromString(rest))
		}
	}

	// example: "some.Type2"
	if index := strings.LastIndex(name, "."); index != -1 {
		result.PkgImportName = name[:index]
		result.PkgName = GuessPkgNameFromPkgImportName(result.PkgImportName)
		result.Name = name[index+1:]
	} else {
		result.Name = name
	}

	return result
}
