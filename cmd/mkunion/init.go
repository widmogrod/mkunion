package main

import (
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
	"golang.org/x/mod/modfile"
)

// templatesFS holds the project skeleton that `mkunion init` writes:
// templates/api is the module (go.mod, AGENTS.md, api/server.go, cmd/server)
// and templates/resource is one resource package (query, command, answer,
// handler, store, endpoint and tests). Every file is a text/template over
// initData, rendered with the .tmpl suffix dropped.
//
//go:embed templates
var templatesFS embed.FS

// initData is what the templates know.
type initData struct {
	Module         string // the Go module path, example.com/todo
	Resource       string // the resource package name, task
	Type           string // the exported type name, Task
	MkunionReplace string // a local path for a replace directive, or ""
	MkunionVersion string // the mkunion version to require, or "" to let go mod tidy pick
}

// resourceNamePattern is a Go package name that is also a URL segment.
var resourceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:        "init",
		Usage:       "Bootstrap an API module or a resource package",
		Description: "Write code that follows the mkunion API conventions, so you define shapes and write handler bodies:\n\n   mkunion init api --module example.com/todo --resource task --dir ./todo\n   mkunion init resource --name project\n\nThe generated AGENTS.md says how to work in the result.",
		Subcommands: []*cli.Command{
			{
				Name:  "api",
				Usage: "Write a new Go module: an HTTP API with one resource, tests, AGENTS.md",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "module", Usage: "Go module path of the new project", Required: true},
					&cli.StringFlag{Name: "resource", Usage: "name of the first resource package", Value: "task"},
					&cli.StringFlag{Name: "dir", Usage: "directory to write the module to; must be empty or missing", Required: true},
					&cli.StringFlag{Name: "mkunion-replace", Usage: "local path to an mkunion checkout, written as a replace directive (for development)"},
					&cli.StringFlag{Name: "mkunion-version", Usage: "mkunion version to require; empty lets go mod tidy pick the latest"},
				},
				Action: initAPIAction,
			},
			{
				Name:      "resource",
				Usage:     "Add a resource package under api/resource/<name> of an existing module",
				ArgsUsage: " [name]",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "resource package name (lowercase, also the URL segment)"},
					&cli.StringFlag{Name: "dir", Usage: "module root, where go.mod is", Value: "."},
				},
				Action: initResourceAction,
			},
		},
	}
}

func newInitData(module, resource, replace, version string) (initData, error) {
	if module == "" {
		return initData{}, fmt.Errorf("init: a module path is required")
	}
	if !resourceNamePattern.MatchString(resource) {
		return initData{}, fmt.Errorf("init: resource name %q must match %s (a Go package name that is also a URL segment)", resource, resourceNamePattern)
	}
	if replace != "" && version == "" {
		version = "v0.0.0-00010101000000-000000000000"
	}
	if replace != "" && !filepath.IsAbs(replace) {
		abs, err := filepath.Abs(replace)
		if err != nil {
			return initData{}, err
		}
		replace = abs
	}
	return initData{
		Module:         module,
		Resource:       resource,
		Type:           strings.ToUpper(resource[:1]) + resource[1:],
		MkunionReplace: replace,
		MkunionVersion: version,
	}, nil
}

func initAPIAction(c *cli.Context) error {
	data, err := newInitData(c.String("module"), c.String("resource"), c.String("mkunion-replace"), c.String("mkunion-version"))
	if err != nil {
		return err
	}
	dir := c.String("dir")
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("init api: %s is not empty", dir)
	}

	written, err := renderTree("templates/api", dir, data)
	if err != nil {
		return err
	}
	more, err := renderTree("templates/resource", filepath.Join(dir, "api", "resource", data.Resource), data)
	if err != nil {
		return err
	}
	written = append(written, more...)

	for _, f := range written {
		fmt.Println(f)
	}
	fmt.Printf("\nWrote %s (%d files). Next:\n\n   cd %s && go mod tidy && go generate ./... && go test ./... -race\n\nThen read AGENTS.md.\n", data.Module, len(written), dir)
	return nil
}

func initResourceAction(c *cli.Context) error {
	name := c.String("name")
	if name == "" {
		name = c.Args().First()
	}
	if name == "" {
		return fmt.Errorf("init resource: a name is required (--name project)")
	}
	root := c.String("dir")
	module, err := modulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	data, err := newInitData(module, name, "", "")
	if err != nil {
		return err
	}
	target := filepath.Join(root, "api", "resource", data.Resource)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("init resource: %s exists", target)
	}

	written, err := renderTree("templates/resource", target, data)
	if err != nil {
		return err
	}
	for _, f := range written {
		fmt.Println(f)
	}
	fmt.Printf(`
Wrote resource %[1]s (%[2]d files). Mount it:

  1. api/server.go, import %[3]q, then in Deps:
         %[4]s *%[1]s.Handler
     and in New:
         mount(s, "%[1]s/query", %[1]s.QueryEndpoint(deps.%[4]s))
         mount(s, "%[1]s/command", %[1]s.CommandEndpoint(deps.%[4]s))
  2. cmd/server/main.go, in Deps:
         %[4]s: &%[1]s.Handler{Store: %[1]s.NewMemoryStore()},
  3. go generate ./... && go test ./... -race
`, data.Resource, len(written), module+"/api/resource/"+data.Resource, data.Type)
	return nil
}

// modulePath reads the module path from a go.mod file.
func modulePath(goMod string) (string, error) {
	content, err := os.ReadFile(goMod)
	if err != nil {
		return "", fmt.Errorf("init: %w (run in the module root or pass --dir)", err)
	}
	module := modfile.ModulePath(content)
	if module == "" {
		return "", fmt.Errorf("init: %s has no module line", goMod)
	}
	return module, nil
}

// renderTree renders every template under root of templatesFS into dir,
// keeping the tree and dropping the .tmpl suffix. Go files are gofmt-ed, so a
// template that renders invalid Go fails here, not in the user's build.
func renderTree(root, dir string, data initData) ([]string, error) {
	var written []string
	err := fs.WalkDir(templatesFS, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, root+"/")
		out := filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(rel, ".tmpl")))

		tmpl, err := template.ParseFS(templatesFS, p)
		if err != nil {
			return fmt.Errorf("init: template %s: %w", p, err)
		}
		var buf strings.Builder
		if err := tmpl.ExecuteTemplate(&buf, path.Base(p), data); err != nil {
			return fmt.Errorf("init: template %s: %w", p, err)
		}
		content := []byte(buf.String())
		if strings.HasSuffix(out, ".go") {
			formatted, err := format.Source(content)
			if err != nil {
				return fmt.Errorf("init: template %s renders invalid Go: %w", p, err)
			}
			content = formatted
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, content, 0o644); err != nil {
			return err
		}
		written = append(written, out)
		return nil
	})
	return written, err
}
