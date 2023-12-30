package generators

import (
	"fmt"
	"sort"
	"strings"
)

type PkgMap = map[string]string
type InitFuncs = []string

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

func GenerateInitFunc(inits InitFuncs) string {
	if len(inits) == 0 {
		return ""
	}

	result := &strings.Builder{}
	result.WriteString("func init() {\n")

	for _, init := range inits {
		result.WriteString(fmt.Sprintf("\t%s\n", init))
	}
	result.WriteString("}\n\n")

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

func padLeftTabs(n int, s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat("\t", n) + line
	}
	return strings.Join(lines, "\n")
}

func padLeftTabs2(n int, s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		lines[i] = strings.Repeat("\t", n) + line
	}
	return strings.Join(lines, "\n")
}
