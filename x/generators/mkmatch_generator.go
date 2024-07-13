package generators

import (
	_ "embed"
	"fmt"
	"strings"
)

type MkMatchGenerator struct {
	Header      string
	PackageName string
	MatchSpecs  []*MatchSpec
	pkgUsed     PkgMap

	skipImportsAndPackage bool
}

func (g *MkMatchGenerator) SkipImportsAndPackage(flag bool) *MkMatchGenerator {
	g.skipImportsAndPackage = flag
	return g
}

func (g *MkMatchGenerator) Generate() ([]byte, error) {
	var sb strings.Builder

	if !g.skipImportsAndPackage {
		// Write the header
		sb.WriteString(g.Header)
		sb.WriteString("\n")

		// Write the package name
		sb.WriteString("package ")
		sb.WriteString(g.PackageName)
		sb.WriteString("\n\n")

		// Write the imports
		g.pkgUsed = make(PkgMap)
		for _, spec := range g.MatchSpecs {
			for _, used := range spec.UsedPackMap {
				g.pkgUsed[used] = used
			}
		}

		impPart := GenerateImports(g.pkgUsed)

		sb.WriteString(impPart)
	}

	for _, spec := range g.MatchSpecs {
		// Function R0
		sb.WriteString("func ")
		sb.WriteString(spec.Name)
		sb.WriteString("R0[")

		for k, t := range spec.Inputs {
			if k != 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("T%d %s", k, t))
		}
		sb.WriteString("](\n")

		for k := range spec.Inputs {
			sb.WriteString(fmt.Sprintf("\tt%d T%d,\n", k, k))
		}

		for k, args := range spec.Cases {
			sb.WriteString(fmt.Sprintf("\tf%d func(", k))
			for i, arg := range args {
				if i != 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("x%d %s", i, arg))
			}
			sb.WriteString("),\n")
		}

		sb.WriteString(") {\n")

		for k, args := range spec.Cases {
			for i, arg := range args {
				sb.WriteString(fmt.Sprintf("\tc%dt%d, c%dt%dok := any(t%d).(%s)\n", k, i, k, i, i, arg))
			}
			sb.WriteString("\tif ")
			for i, _ := range args {
				if i != 0 {
					sb.WriteString(" && ")
				}
				sb.WriteString(fmt.Sprintf("c%dt%dok", k, i))
			}
			sb.WriteString(" {\n\t\tf")
			sb.WriteString(fmt.Sprintf("%d(", k))
			for i := range args {
				if i != 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("c%dt%d", k, i))
			}
			sb.WriteString(")\n\t\treturn\n\t}\n")
		}
		sb.WriteString(fmt.Sprintf("\tpanic(\"%sR0 is not exhaustive\")\n}\n\n", spec.Name))

		// Functions R1, R2, R3
		for returnTypes := 1; returnTypes <= 3; returnTypes++ {
			sb.WriteString(fmt.Sprintf("func %sR%d[", spec.Name, returnTypes))

			for k, t := range spec.Inputs {
				if k != 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("T%d %s", k, t))
			}

			for o := 1; o <= returnTypes; o++ {
				sb.WriteString(fmt.Sprintf(", TOut%d any", o))
			}

			sb.WriteString("](\n")

			for k := range spec.Inputs {
				sb.WriteString(fmt.Sprintf("\tt%d T%d,\n", k, k))
			}

			for k, args := range spec.Cases {
				sb.WriteString(fmt.Sprintf("\tf%d func(", k))
				for i, arg := range args {
					if i != 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(fmt.Sprintf("x%d %s", i, arg))
				}
				sb.WriteString(") (")
				for o := 1; o <= returnTypes; o++ {
					if o != 1 {
						sb.WriteString(", ")
					}
					sb.WriteString(fmt.Sprintf("TOut%d", o))
				}
				sb.WriteString("),\n")
			}

			sb.WriteString(") (")

			for o := 1; o <= returnTypes; o++ {
				if o != 1 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("TOut%d", o))
			}

			sb.WriteString(") {\n")

			for k, args := range spec.Cases {
				for i, arg := range args {
					sb.WriteString(fmt.Sprintf("\tc%dt%d, c%dt%dok := any(t%d).(%s)\n", k, i, k, i, i, arg))
				}
				sb.WriteString("\tif ")
				for i, _ := range args {
					if i != 0 {
						sb.WriteString(" && ")
					}
					sb.WriteString(fmt.Sprintf("c%dt%dok", k, i))
				}
				sb.WriteString(" {\n\t\treturn f")
				sb.WriteString(fmt.Sprintf("%d(", k))
				for i := range args {
					if i != 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(fmt.Sprintf("c%dt%d", k, i))
				}
				sb.WriteString(")\n\t}\n")
			}
			sb.WriteString(fmt.Sprintf("\tpanic(\"%sR%d is not exhaustive\")\n}\n\n", spec.Name, returnTypes))
		}
	}

	return []byte(sb.String()), nil
}
