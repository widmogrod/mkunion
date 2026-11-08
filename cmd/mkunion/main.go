//go:tag mkunion:",no-type-registry"
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/widmogrod/mkunion/cmd/internal/gen"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shape/breaking"
	"github.com/widmogrod/mkunion/x/shared"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	log.SetOutput(os.Stderr)
	log.SetLevel(log.ErrorLevel)

	app := newApp(ctx)

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func newApp(ctx context.Context) *cli.App {
	return &cli.App{
		Name:                   shared.Program,
		Usage:                  "Strongly typed union types in Go",
		Description:            "Generates union types, shapes, JSON serde and pattern matching from //go:tag mkunion:\"...\" declarations.\nRun without a command to generate code for specific files (used in //go:generate directives),\nor use 'watch' to generate for whole packages.",
		EnableBashCompletion:   true,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "union name(s) to generate; when not provided, all unions in the input files are generated",
				Required: false,
			},
			inputGoFileFlag(`Go file(s) to generate code for; when not provided, falls back to the GOFILE environment variable, used when combined with //go:tag mkunion:"MyUnionName"`),
			verboseFlag(),
			&cli.BoolFlag{
				Name:  "type-registry",
				Usage: "generate types_reg_gen.go type registry, used for JSON serde of generic types",
				Value: true,
			},
		},
		Action: generateAction,
		Commands: []*cli.Command{
			{
				Name:        "shape-export",
				Usage:       "Export Go types to other languages (TypeScript, Swift)",
				ArgsUsage:   " [file.go ...]",
				Description: "Generate TypeScript or Swift types from Go types, and enable end-to-end type safety.\nFiles come from -i, positional arguments, or GOFILE under go generate:\n\n   mkunion shape-export --language swift -o ../Sources/Api api.go",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "language",
						Aliases: []string{"lang"},
						Usage:   "output language: typescript or swift",
						Value:   "typescript",
					},
					&cli.StringFlag{
						Name:    "output-dir",
						Aliases: []string{"o", "output"},
						Usage:   "directory to write exported types to",
					},
					inputGoFileFlag(`Go file(s) to export types from; when not provided, falls back to the GOFILE environment variable`),
					verboseFlag(),
				},
				Action: shapeExportAction,
			},
			{
				Name:        "shape-snapshot",
				Usage:       "Save the shapes of an API, for 'mkunion breaking'",
				ArgsUsage:   " [file.go ...]",
				Description: "Write every union, variant, struct and alias of the given Go files as one JSON document.\nCommit it next to the API and compare later changes against it with 'mkunion breaking':\n\n   mkunion shape-snapshot -o shapes.json api.go",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "file to write the snapshot to",
						Value:   "shapes.json",
					},
					inputGoFileFlag(`Go file(s) to snapshot; when not provided, falls back to the GOFILE environment variable`),
					verboseFlag(),
				},
				Action: shapeSnapshotAction,
			},
			{
				Name:        "breaking",
				Usage:       "List wire-breaking changes against a shape snapshot",
				ArgsUsage:   " [file.go ...]",
				Description: "Infer the shapes of the given Go files, compare them with a snapshot from 'mkunion shape-snapshot'\nand print one finding per line as file:line: severity: subject: message.\nExits 1 when any finding is breaking for clients built against the snapshot:\n\n   mkunion breaking --against shapes.json api.go",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "against",
						Usage:    "snapshot written by shape-snapshot",
						Required: true,
					},
					inputGoFileFlag(`Go file(s) to check; when not provided, falls back to the GOFILE environment variable`),
					verboseFlag(),
				},
				Action: breakingAction,
			},
			{
				Name:        "watch",
				Usage:       "Generate code for packages and watch for changes",
				ArgsUsage:   " [./... | paths]",
				Description: "Generate union types and shapes for all packages under the given paths,\nrun 'go generate ./...', and keep watching for file changes.\nUse -g for one-time generation (CI, scripts):\n\n   mkunion watch -g ./...",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "type-registry",
						Usage: "generate types_reg_gen.go type registries, used for JSON serde of generic types",
						Value: true,
					},
					verboseFlag(),
					&cli.BoolFlag{
						Name:     "generate-only",
						Aliases:  []string{"g"},
						Usage:    "generate once and exit, instead of watching for changes",
						Required: false,
						Value:    false,
					},
					&cli.BoolFlag{
						Name:     "dont-run-go-generate",
						Aliases:  []string{"G"},
						Required: false,
						Value:    false,
						Usage:    "skip running 'go generate ./...' after mkunion generation",
					},
				},
				Action: func(c *cli.Context) error {
					return watchAction(ctx, c)
				},
			},
			{
				Name:        "clean",
				Usage:       "Remove all generated files",
				ArgsUsage:   " [./... | paths]",
				Description: "Remove all mkunion-generated files to have clean state for generation.\nA file is removed only when its name matches a generated pattern\n(*_union_gen.go, *_shape_gen.go, *_serde_gen.go, *_match_gen.go, types_reg_gen.go)\nand its first line is the '// Code generated by mkunion. DO NOT EDIT.' header.\nPreview with -n:\n\n   mkunion clean -n ./...",
				Flags: []cli.Flag{
					verboseFlag(),
					&cli.BoolFlag{
						Name:     "dry-run",
						Aliases:  []string{"n"},
						Required: false,
						Value:    false,
						Usage:    "Show what files would be removed without actually removing them",
					},
				},
				Action: cleanAction,
			},
			initCommand(),
		},
	}
}

func verboseFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:     "verbose",
		Aliases:  []string{"v"},
		Usage:    "enable debug-level logging",
		Required: false,
		Value:    false,
	}
}

func inputGoFileFlag(usage string) cli.Flag {
	return &cli.StringSliceFlag{
		Name:      "input-go-file",
		Aliases:   []string{"i", "input"},
		Usage:     usage,
		TakesFile: true,
	}
}

func setLogLevel(c *cli.Context, defaultLevel log.Level) {
	log.SetLevel(defaultLevel)
	if c.Bool("verbose") {
		log.SetLevel(log.DebugLevel)
	}
}

// sourcePathsOrExit resolves the input files from -i flags, positional
// arguments or GOFILE, and shows the command help when there are none.
func sourcePathsOrExit(c *cli.Context) []string {
	inputs := append(c.StringSlice("input-go-file"), c.Args().Slice()...)
	sourcePaths := gen.SourcePathsOrGOFILE(inputs)
	if len(sourcePaths) == 0 {
		cli.ShowSubcommandHelpAndExit(c, 1)
	}
	return sourcePaths
}

func generateAction(c *cli.Context) error {
	setLogLevel(c, log.ErrorLevel)

	sourcePaths := sourcePathsOrExit(c)

	savedFiles, err := gen.Generate(sourcePaths, c.Bool("type-registry"))
	if err != nil {
		return err
	}

	for _, x := range savedFiles {
		fmt.Println(x)
	}

	return nil
}

func shapeExportAction(c *cli.Context) error {
	setLogLevel(c, log.ErrorLevel)

	sourcePaths := sourcePathsOrExit(c)

	var renderer interface {
		AddShape(shape.Shape)
		WriteToDir(string) error
	}
	switch lang := c.String("language"); lang {
	case "typescript", "ts":
		renderer = shape.NewTypeScriptRenderer()
	case "swift":
		renderer = shape.NewSwiftRenderer()
	default:
		return fmt.Errorf("shape-export: unknown language %q; use typescript or swift", lang)
	}

	for _, sourcePath := range sourcePaths {
		inferred, err := shape.InferFromFile(sourcePath)
		if err != nil {
			return err
		}

		for _, x := range inferred.RetrieveShapes() {
			renderer.AddShape(x)
		}
	}

	err := renderer.WriteToDir(c.String("output-dir"))
	if err != nil {
		return fmt.Errorf("failed to write to dir %s: %w", c.String("output-dir"), err)
	}

	return nil
}

func shapeSnapshotAction(c *cli.Context) error {
	setLogLevel(c, log.ErrorLevel)

	sourcePaths := sourcePathsOrExit(c)

	snapshot, err := breaking.Take(sourcePaths)
	if err != nil {
		return err
	}

	err = breaking.WriteFile(c.String("output"), snapshot)
	if err != nil {
		return fmt.Errorf("shape-snapshot: %w", err)
	}

	fmt.Fprintln(c.App.Writer, c.String("output"))
	return nil
}

func breakingAction(c *cli.Context) error {
	setLogLevel(c, log.ErrorLevel)

	sourcePaths := sourcePathsOrExit(c)

	against, err := breaking.ReadFile(c.String("against"))
	if err != nil {
		return err
	}

	current, err := breaking.Take(sourcePaths)
	if err != nil {
		return err
	}

	loc, err := breaking.Locate(sourcePaths)
	if err != nil {
		return fmt.Errorf("breaking: %w", err)
	}

	findings := breaking.Compare(against, current, loc)
	for _, finding := range findings {
		fmt.Fprintln(c.App.Writer, finding.String())
	}

	if n := breaking.CountBreaking(findings); n > 0 {
		return cli.Exit(fmt.Sprintf("breaking: %d breaking change(s)", n), 1)
	}
	return nil
}

func watchAction(ctx context.Context, c *cli.Context) error {
	setLogLevel(c, log.InfoLevel)

	paths, err := gen.ExpandPathArgs(c.Args().Slice())
	if err != nil {
		return err
	}

	// generate first before watching
	sourcePaths, err := gen.GoFilesFromDirs(paths)
	if err != nil {
		return fmt.Errorf("extracting source paths: %w", err)
	}

	log.Printf("Initial generation for %d go files", len(sourcePaths))

	prevLevel := log.GetLevel()
	log.SetLevel(log.ErrorLevel)
	savedFiles, err := gen.Generate(sourcePaths, c.Bool("type-registry"))
	log.SetLevel(prevLevel)
	if err != nil {
		return fmt.Errorf("initial generation: %w", err)
	}

	for _, x := range savedFiles {
		log.Infof("   generated:\t%s", x)
	}

	if !c.Bool("dont-run-go-generate") {
		// Errors are already logged by RunGoGenerate
		_ = gen.RunGoGenerate(paths)
	}

	if os.Getenv("MKUNION_STATS") != "" {
		for k, v := range shape.DefaultIndex.Stats() {
			fmt.Fprintf(os.Stderr, "stats: %s=%d\n", k, v)
		}
	}

	if c.Bool("generate-only") {
		return nil
	}

	log.Printf("Watching for changes ...")
	return gen.Watch(ctx, paths, gen.WatchOptions{
		TypeRegistry:  c.Bool("type-registry"),
		RunGoGenerate: !c.Bool("dont-run-go-generate"),
	})
}

func cleanAction(c *cli.Context) error {
	setLogLevel(c, log.ErrorLevel)

	paths, err := gen.ExpandPathArgs(c.Args().Slice())
	if err != nil {
		return err
	}

	removedFiles, err := gen.CleanGeneratedFiles(paths, c.Bool("dry-run"))
	if err != nil {
		return err
	}

	if len(removedFiles) == 0 {
		fmt.Println("No generated files found to remove")
		return nil
	}

	if c.Bool("dry-run") {
		fmt.Printf("Would remove %d files:\n", len(removedFiles))
	} else {
		fmt.Printf("Removed %d generated files:\n", len(removedFiles))
	}
	for _, file := range removedFiles {
		fmt.Printf("  %s\n", file)
	}

	return nil
}
