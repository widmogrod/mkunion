package generators

import (
	"fmt"
	"sort"
	"strings"
)

type PkgMap = map[string]string

func GenerateImports(pkgMap PkgMap) string {
	if len(pkgMap) == 0 {
		return ""
	}

	result := &strings.Builder{}
	result.WriteString("import (\n")

	var sortedImportNames []string
	for _, pkgImportName := range pkgMap {
		sortedImportNames = append(sortedImportNames, pkgImportName)
	}
	sort.Strings(sortedImportNames)

	for _, pkgImportName := range sortedImportNames {
		result.WriteString(fmt.Sprintf("\t\"%s\"\n", pkgImportName))
	}
	result.WriteString(")\n\n")

	return result.String()

}

func MergePkgMaps(maps ...PkgMap) PkgMap {
	result := make(PkgMap)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
