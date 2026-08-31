package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/tools/cover"
)

// crapScore implements the CRAP (Change Risk Anti-Patterns) metric:
//
//	CRAP(fn) = comp^2 * (1 - coverage)^3 + comp
//
// where comp is cyclomatic complexity and coverage is the fraction
// of statements in the function executed by tests (0..1).
func crapScore(complexity int, coverage float64) float64 {
	c := float64(complexity)
	return c*c*math.Pow(1-coverage, 3) + c
}

type funcStat struct {
	Name       string
	File       string
	Line       int
	Complexity int
	Coverage   float64
	Crap       float64
}

func main() {
	app := &cli.App{
		Name:        "mkcrap",
		Description: "mkcrap computes the CRAP metric (cyclomatic complexity vs test coverage) per function and fails when any function exceeds the threshold",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "profile",
				Value: "coverage.out",
				Usage: "coverage profile produced by 'go test -coverprofile'",
			},
			&cli.Float64Flag{
				Name:  "threshold",
				Value: 30,
				Usage: "fail when any function has a CRAP score above this value",
			},
			&cli.IntFlag{
				Name:  "top",
				Value: 20,
				Usage: "number of worst offenders to print",
			},
			&cli.BoolFlag{
				Name:  "include-generated",
				Value: false,
				Usage: "include files marked '// Code generated ... DO NOT EDIT.'",
			},
			&cli.StringSliceFlag{
				Name:  "skip",
				Usage: "path prefixes to skip (relative to module root), can be repeated",
			},
		},
		Action: func(c *cli.Context) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}

			modulePath, err := readModulePath(filepath.Join(root, "go.mod"))
			if err != nil {
				return fmt.Errorf("mkcrap must run from the module root: %w", err)
			}

			profiles, err := cover.ParseProfiles(c.String("profile"))
			if err != nil {
				return fmt.Errorf("cannot read coverage profile %q (run 'go test -coverprofile=%s ./...' first): %w",
					c.String("profile"), c.String("profile"), err)
			}

			blocksByFile := map[string][]cover.ProfileBlock{}
			for _, p := range profiles {
				blocksByFile[p.FileName] = append(blocksByFile[p.FileName], p.Blocks...)
			}

			stats, err := collectStats(root, modulePath, blocksByFile, c.Bool("include-generated"), c.StringSlice("skip"))
			if err != nil {
				return err
			}

			sort.Slice(stats, func(i, j int) bool { return stats[i].Crap > stats[j].Crap })

			threshold := c.Float64("threshold")

			var failing []funcStat
			for _, s := range stats {
				if s.Crap > threshold {
					failing = append(failing, s)
				}
			}

			top := c.Int("top")
			if top > len(stats) {
				top = len(stats)
			}
			fmt.Printf("%-8s %-6s %-6s  %s\n", "CRAP", "CC", "COV%", "FUNCTION")
			for _, s := range stats[:top] {
				marker := " "
				if s.Crap > threshold {
					marker = "!"
				}
				fmt.Printf("%-8.1f %-6d %-6.1f %s%s (%s:%d)\n",
					s.Crap, s.Complexity, s.Coverage*100, marker, s.Name, s.File, s.Line)
			}

			if len(failing) > 0 {
				fmt.Printf("\nFAIL: %d function(s) exceed CRAP threshold %.1f:\n", len(failing), threshold)
				for _, s := range failing {
					fmt.Printf("  %.1f %s (%s:%d)\n", s.Crap, s.Name, s.File, s.Line)
				}
				return cli.Exit("add tests or reduce complexity (see list above)", 1)
			}
			fmt.Printf("\nOK: all %d functions are at or below CRAP threshold %.1f\n", len(stats), threshold)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		if _, ok := err.(cli.ExitCoder); ok {
			cli.HandleExitCoder(err)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func readModulePath(gomod string) (string, error) {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no module directive in %s", gomod)
}

var generatedRe = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

func collectStats(root, modulePath string, blocksByFile map[string][]cover.ProfileBlock, includeGenerated bool, skip []string) ([]funcStat, error) {
	var stats []funcStat
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			name := d.Name()
			if rel != "." && (strings.HasPrefix(name, ".") || name == "testdata" || name == "node_modules") {
				return filepath.SkipDir
			}
			for _, s := range skip {
				if rel == strings.TrimSuffix(s, "/") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if !includeGenerated && isGenerated(file) {
			return nil
		}

		coverName := modulePath + "/" + filepath.ToSlash(rel)
		blocks := blocksByFile[coverName]

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			start := fset.Position(fn.Pos())
			end := fset.Position(fn.End())

			covered, total := 0, 0
			for _, b := range blocks {
				if b.StartLine >= start.Line && b.EndLine <= end.Line {
					total += b.NumStmt
					if b.Count > 0 {
						covered += b.NumStmt
					}
				}
			}
			coverage := 0.0
			if total > 0 {
				coverage = float64(covered) / float64(total)
			}

			comp := complexity(fn)
			stats = append(stats, funcStat{
				Name:       funcName(fn),
				File:       rel,
				Line:       start.Line,
				Complexity: comp,
				Coverage:   coverage,
				Crap:       crapScore(comp, coverage),
			})
		}
		return nil
	})
	return stats, err
}

func isGenerated(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			break
		}
		for _, comment := range group.List {
			if generatedRe.MatchString(comment.Text) {
				return true
			}
		}
	}
	return false
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		var buf strings.Builder
		writeRecvType(&buf, fn.Recv.List[0].Type)
		return buf.String() + "." + fn.Name.Name
	}
	return fn.Name.Name
}

func writeRecvType(buf *strings.Builder, expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.StarExpr:
		buf.WriteString("*")
		writeRecvType(buf, t.X)
	case *ast.Ident:
		buf.WriteString(t.Name)
	case *ast.IndexExpr:
		writeRecvType(buf, t.X)
	case *ast.IndexListExpr:
		writeRecvType(buf, t.X)
	default:
		buf.WriteString("?")
	}
}

// complexity is gocyclo-style cyclomatic complexity: 1 + one for each
// branching construct (if, for, range, non-default case, &&, ||).
func complexity(fn *ast.FuncDecl) int {
	comp := 1
	ast.Inspect(fn, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			comp++
		case *ast.CaseClause:
			if n.List != nil {
				comp++
			}
		case *ast.CommClause:
			if n.Comm != nil {
				comp++
			}
		case *ast.BinaryExpr:
			if n.Op == token.LAND || n.Op == token.LOR {
				comp++
			}
		}
		return true
	})
	return comp
}
