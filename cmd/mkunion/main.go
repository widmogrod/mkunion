//go:tag mkunion:",no-type-registry"
package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/widmogrod/mkunion/x/generators"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// set log level to error
	log.SetOutput(os.Stderr)
	log.SetLevel(log.ErrorLevel)

	var app *cli.App
	app = &cli.App{
		Name:                   shared.Program,
		Description:            "Strongly typed union type in golang.",
		EnableBashCompletion:   true,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:      "input-go-file",
				Aliases:   []string{"i", "input"},
				Usage:     `When not provided, it will try to use GOFILE environment variable, used when combined with //go:tag mkunion:"MyNunionName"`,
				TakesFile: true,
			},
			&cli.BoolFlag{
				Name:     "verbose",
				Aliases:  []string{"v"},
				Required: false,
				Value:    false,
			},
			&cli.BoolFlag{
				Name:  "type-registry",
				Value: true,
			},
		},
		Action: func(c *cli.Context) error {
			if c.Bool("verbose") {
				log.SetLevel(log.DebugLevel)
			}

			sourcePaths := c.StringSlice("input-go-file")
			if len(sourcePaths) == 0 && os.Getenv("GOFILE") != "" {
				cwd, _ := syscall.Getwd()
				sourceName := path.Base(os.Getenv("GOFILE"))
				sourcePaths = []string{
					path.Join(cwd, sourceName),
				}
			}

			if len(sourcePaths) == 0 {
				// show usage
				cli.ShowAppHelpAndExit(c, 1)
			}

			savedFiles, err := GenerateMain(sourcePaths, c.Bool("type-registry"))
			if err != nil {
				return err
			}

			for _, x := range savedFiles {
				fmt.Println(x)
			}

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:        "shape-export",
				Description: "Generate typescript types from golang types, and enable end-to-end type safety.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "language",
						Aliases:     []string{"lang"},
						DefaultText: "typescript",
					},
					&cli.StringFlag{
						Name:    "output-dir",
						Aliases: []string{"o", "output"},
					},
					&cli.StringSliceFlag{
						Name:      "input-go-file",
						Aliases:   []string{"i", "input"},
						Usage:     `When not provided, it will try to use GOFILE environment variable, used when combined with //go:tag mkunion:"MyNunionName"`,
						TakesFile: true,
					},
					&cli.BoolFlag{
						Name:     "verbose",
						Aliases:  []string{"v"},
						Required: false,
						Value:    false,
					},
				},
				Action: func(c *cli.Context) error {
					if c.Bool("verbose") {
						log.SetLevel(log.DebugLevel)
					}

					sourcePaths := c.StringSlice("input-go-file")
					if len(sourcePaths) == 0 && os.Getenv("GOFILE") != "" {
						cwd, _ := syscall.Getwd()
						sourceName := path.Base(os.Getenv("GOFILE"))
						sourcePaths = []string{
							path.Join(cwd, sourceName),
						}
					}

					if len(sourcePaths) == 0 {
						// show usage
						cli.ShowAppHelpAndExit(c, 1)
					}

					tsr := shape.NewTypeScriptRenderer()
					for _, sourcePath := range sourcePaths {
						// file name without extension
						inferred, err := shape.InferFromFile(sourcePath)
						if err != nil {
							return err
						}

						for _, x := range inferred.RetrieveShapes() {
							tsr.AddShape(x)
						}
					}

					err := tsr.WriteToDir(c.String("output-dir"))
					if err != nil {
						return fmt.Errorf("failed to write to dir %s: %w", c.String("output-dir"), err)
					}

					return nil
				},
			},
			{
				Name:        "watch",
				Description: "Watch for changes in the directory and get mkunion generative features instantly",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "type-registry",
						Value: true,
					},
					&cli.BoolFlag{
						Name:     "verbose",
						Aliases:  []string{"v"},
						Required: false,
						Value:    false,
					},
					&cli.BoolFlag{
						Name:     "generate-only",
						Aliases:  []string{"g"},
						Required: false,
						Value:    false,
					},
					&cli.BoolFlag{
						Name:     "dont-run-go-generate",
						Aliases:  []string{"G"},
						Required: false,
						Value:    false,
						Usage:    "Skip running 'go generate ./...' after mkunion generation (default: false)",
					},
				},
				Action: func(c *cli.Context) error {
					log.SetLevel(log.InfoLevel)
					if c.Bool("verbose") {
						log.SetLevel(log.DebugLevel)
					}

					// Create a new watcher
					watcher, err := fsnotify.NewWatcher()
					if err != nil {
						log.Fatal(err)
					}
					defer watcher.Close()

					// Create a channel to receive events
					done := make(chan bool)

					justGenerated := sync.Map{}
					justChanged := sync.Map{}
					justRemoved := sync.Map{}
					dontRunGoGenerate := c.Bool("dont-run-go-generate")

					// Start a goroutine to handle events
					go func() {
						defer close(done)
						for {
							select {
							case <-ctx.Done():
								return

							case event, ok := <-watcher.Events:
								if !ok {
									return
								}

								// is .go file?
								if filepath.Ext(event.Name) != ".go" {
									continue
								}

								if event.Op&fsnotify.Chmod == fsnotify.Chmod {
									continue
								}

								// extract path name from event.Name
								pathName := strings.Trim(event.Name, `"`)
								if event.Op&fsnotify.Remove == fsnotify.Remove {
									justRemoved.Store(pathName, true)
								} else {
									justChanged.Store(pathName, true)
								}

								// if the file was generated by watch process, skip it
								if _, ok := justGenerated.Load(pathName); ok {
									// but to prevent removing it to fast and resulting in infinit-generation loop
									// 1s debounce is applied
									go func(pathName string) {
										time.Sleep(2 * time.Second)
										justGenerated.Delete(pathName)
									}(pathName)
									continue
								}

							case err, ok := <-watcher.Errors:
								if !ok {
									return
								}
								log.Errorf("Error: %s", err)

							case <-time.After(250 * time.Millisecond):
								// try to proces deleted files first
								deletedUnique := make(map[string]bool)
								// if there is no event for 1s, check if there is any file removed
								justRemoved.Range(func(key, value interface{}) bool {
									pathName := key.(string)
									dir := filepath.Dir(pathName)
									if _, ok := justGenerated.Load(pathName); !ok {
										deletedUnique[dir] = true
									}
									justRemoved.Delete(key)
									return true
								})

								deleted := make([]string, 0, len(deletedUnique))
								for x := range deletedUnique {
									deleted = append(deleted, x)
								}

								if len(deleted) > 0 {
									prevLevel := log.GetLevel()
									log.SetLevel(log.ErrorLevel)
									savedFiles, err := GenerateTypeRegistryForDir(deleted)
									log.SetLevel(prevLevel)

									if err != nil {
										log.Warnf("failed to generate type registry for %v: %s", deleted, err)
										continue
									}

									for _, x := range savedFiles {
										log.Infof("re-generated:\t%s", x)
										justGenerated.Store(x, true)
									}

									if len(savedFiles) > 0 {
										log.Infof("re-generated: done for deleted files")
									} else {
										log.Infof("re-generated: delete not resulted in any new files")
									}
								}

								// now process changed files
								var changedSourcePaths []string
								justChanged.Range(func(key, value interface{}) bool {
									pathName := key.(string)
									if _, ok := justGenerated.Load(pathName); !ok {
										changedSourcePaths = append(changedSourcePaths, pathName)
									}

									justChanged.Delete(key)
									return true
								})

								if len(changedSourcePaths) > 0 {
									prevLevel := log.GetLevel()
									log.SetLevel(log.ErrorLevel)
									savedFiles, err := GenerateMain(changedSourcePaths, c.Bool("type-registry"))
									log.SetLevel(prevLevel)

									for _, x := range savedFiles {
										log.Infof("re-generated:\t%s", x)
										justGenerated.Store(x, true)
									}

									if err != nil {
										log.Errorf("failed to generate: %s", err)
									}

									if len(savedFiles) > 0 {
										log.Infof("re-generated: done for changed files")

										// Run go generate if not disabled and there were changes
										if !dontRunGoGenerate {
											// Extract directories from changed files
											dirsMap := make(map[string]bool)
											for _, file := range changedSourcePaths {
												dir := filepath.Dir(file)
												dirsMap[dir] = true
											}

											dirs := make([]string, 0, len(dirsMap))
											for dir := range dirsMap {
												dirs = append(dirs, dir)
											}

											// Errors are already logged by runGoGenerate
											_ = runGoGenerate(dirs)
										}
									} else {
										log.Infof("re-generated: change not resulted in any new files")
									}
								}
							}
						}
					}()

					paths := c.Args().Slice()
					if len(paths) == 0 {
						paths = []string{"."}
					}

					paths, err = mapf(paths, artToPath)
					if err != nil {
						return err
					}

					paths = dedup(paths)
					// generate first before watching
					sourcePaths, err := goFilesFromDirs(paths)
					if err != nil {
						return fmt.Errorf("extracting source paths: %w", err)
					}

					log.Printf("Initial generation for %d go files", len(sourcePaths))

					prevLevel := log.GetLevel()
					log.SetLevel(log.ErrorLevel)
					savedFiles, err := GenerateMain(sourcePaths, c.Bool("type-registry"))
					log.SetLevel(prevLevel)
					if err != nil {
						return fmt.Errorf("initial generation: %w", err)
					}

					for _, x := range savedFiles {
						log.Infof("   generated:\t%s", x)
					}

					// Run go generate if not disabled
					if !c.Bool("dont-run-go-generate") {
						// Errors are already logged by runGoGenerate
						_ = runGoGenerate(paths)
					}

					if c.Bool("generate-only") {
						return nil
					}

					log.Printf("Watching for changes ...")
					// Add a directory or file to be watched
					for _, path := range paths {
						err = watcher.Add(path)
						if err != nil {
							return err
						}
					}

					// Block until done is received
					<-done

					return nil
				},
			},
		},
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func dedup(in []string) []string {
	m := make(map[string]bool)
	for _, x := range in {
		m[x] = true
	}

	out := make([]string, 0, len(m))
	for x := range m {
		out = append(out, x)
	}

	return out
}

func mapf(in []string, f func(string) ([]string, error)) ([]string, error) {
	result := make([]string, 0, len(in))
	for _, x := range in {
		y, err := f(x)
		if err != nil {
			return nil, err
		}

		result = append(result, y...)
	}

	return result, nil
}

var (
	ErrUseWorkingDirectory = fmt.Errorf("cannot retrive path of working directory")
	ErrFindingRecursiveDir = fmt.Errorf("failed to find recursive directories")
)

func cwdOrPath(x string) (string, error) {
	if x == "." {
		cwd, err := syscall.Getwd()
		if err != nil {
			return ".", fmt.Errorf("cwdOrPath: %s; %w", x, ErrUseWorkingDirectory)
		}

		return cwd, nil
	}

	return x, nil
}
func artToPath(x string) ([]string, error) {
	switch x {
	case ".":
		cwd, err := cwdOrPath(x)
		if err != nil {
			return nil, fmt.Errorf("artToPath: %s; %w", x, err)
		}

		return []string{cwd}, nil

	default:
		if path.Base(x) == "..." {
			var result []string
			// recursively walk through all directories starting from the directory
			dir, err := cwdOrPath(path.Dir(x))
			if err != nil {
				return nil, fmt.Errorf("artToPath: recursive %s; %w", x, err)
			}

			err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if !d.IsDir() {
					return nil
				}

				// if is hidden directory, skip
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}

				result = append(result, path)
				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("artToPath: %s; %w; %w", x, err, ErrFindingRecursiveDir)
			}

			return result, nil
		}

		return []string{x}, nil
	}
}

func goFilesFromDirs(dirs []string) ([]string, error) {
	var result []string
	for _, dir := range dirs {
		// recursively walk through all directories starting from the directory
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if dir != path && d.IsDir() {
				return filepath.SkipDir
			}

			// if is hidden directory, skip
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}

			// if is not go file, skip
			if filepath.Ext(path) != ".go" {
				return nil
			}

			// if is generated file, skip
			if strings.HasSuffix(path, "_gen.go") {
				return nil
			}

			result = append(result, path)
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("gofilesFromDirs: %w", err)
		}
	}

	return result, nil

}

func GenerateMain(sourcePaths []string, typeRegistry bool) ([]string, error) {
	packages := make(map[string]*shape.InferredInfo)
	var savedFiles []string

	for _, sourcePath := range sourcePaths {
		// file name without extension
		inferred, err := shape.InferFromFile(sourcePath)
		if err != nil {
			return savedFiles, err
		}

		if _, ok := packages[inferred.PackageImportName()]; !ok {
			packages[inferred.PackageImportName()] = inferred
		}

		contents, err := GenerateUnions(inferred)
		if err != nil {
			return savedFiles, fmt.Errorf("failed generating union in %s: %w", sourcePath, err)
		}
		savedFile, err := SaveFile(contents, sourcePath, "union_gen")
		if err != nil {
			return savedFiles, fmt.Errorf("failed saving union in %s: %w", sourcePath, err)
		}
		if len(savedFile) > 0 {
			savedFiles = append(savedFiles, savedFile)
		}

		contents, err = GenerateSerde(inferred)
		if err != nil {
			return savedFiles, fmt.Errorf("failed generating serde in %s: %w", sourcePath, err)
		}
		savedFile, err = SaveFile(contents, sourcePath, "serde_gen")
		if err != nil {
			return savedFiles, fmt.Errorf("failed saving serde in %s: %w", sourcePath, err)
		}
		if len(savedFile) > 0 {
			savedFiles = append(savedFiles, savedFile)
		}

		contents, err = GenerateShape(inferred)
		if err != nil {
			return savedFiles, fmt.Errorf("failed generating shape in %s: %w", sourcePath, err)
		}
		savedFile, err = SaveFile(contents, sourcePath, "shape_gen")
		if err != nil {
			return savedFiles, fmt.Errorf("failed saving shape in %s: %w", sourcePath, err)
		}
		if len(savedFile) > 0 {
			savedFiles = append(savedFiles, savedFile)
		}

		contents, err = GenerateMatch(inferred)
		if err != nil {
			return savedFiles, fmt.Errorf("failed generating match in %s: %w", sourcePath, err)
		}
		savedFile, err = SaveFile(contents, sourcePath, "match_gen")
		if err != nil {
			return savedFiles, fmt.Errorf("failed saving match in %s: %w", sourcePath, err)
		}
		if len(savedFile) > 0 {
			savedFiles = append(savedFiles, savedFile)
		}
	}

	if typeRegistry {
		uniqueDirs := make(map[string]bool)
		for _, inferred := range packages {
			dir := path.Dir(inferred.FileName())
			uniqueDirs[dir] = true
		}

		var dirs []string
		for dir := range uniqueDirs {
			dirs = append(dirs, dir)
		}

		savedFiles2, err := GenerateTypeRegistryForDir(dirs)
		if err != nil {
			return savedFiles, err
		}
		savedFiles = append(savedFiles, savedFiles2...)
	}

	return savedFiles, nil
}

func GenerateTypeRegistryForDir(uniqueDirs []string) ([]string, error) {
	var savedFiles []string
	for _, dir := range uniqueDirs {
		// walk through all *.go files in the same directory
		// and generate type registry for all inferred packages
		// in the same directory

		indexed, err := shape.NewIndexTypeInDir(dir)
		if err != nil {
			return savedFiles, fmt.Errorf("mkunion: failed indexing types in directory %s: %w", dir, err)
		}

		if shape.TagHasOption(indexed.PackageTags(), shape.TagUnionName, shape.TagUnionOptionNoRegistry) {
			continue
		}

		if len(indexed.IndexedShapes()) < 1 {
			continue
		}

		contents, err := GenerateTypeRegistry(indexed)
		if err != nil {
			return savedFiles, fmt.Errorf("mkunion: failed walking through directory %s: %w", dir, err)
		}

		regPath := path.Join(dir, "types.go")
		savedFile, err := SaveFile(contents, regPath, "reg_gen")
		if err != nil {
			return savedFiles, fmt.Errorf("mkunion: failed saving type registry in %s: %w", regPath, err)
		}

		if len(savedFile) > 0 {
			savedFiles = append(savedFiles, savedFile)
		}
	}

	return savedFiles, nil
}

func GenerateUnions(inferred *shape.InferredInfo) (bytes.Buffer, error) {
	shapesContents := bytes.Buffer{}
	unions := inferred.RetrieveUnions()
	if len(unions) == 0 {
		return shapesContents, nil
	}

	var err error
	packageName := "main"
	pkgMap := make(generators.PkgMap)
	initFunc := make(generators.InitFuncs, 0)

	for _, union := range unions {
		packageName = shape.ToGoPkgName(union)

		genVisitor := generators.NewVisitorGenerator(union)
		genVisitor.SkipImportsAndPackage(true)

		contents := []byte("//union:genVisitor\n")
		contents, err = genVisitor.Generate()
		if err != nil {
			return shapesContents, fmt.Errorf("failed to generate genVisitor for %s: %w", shape.ToGoTypeName(union), err)
		}
		shapesContents.Write(contents)

		if shape.TagHasOption(union.Tags, "mkunion", "noserde") {
			continue
		}

		genSerde := generators.NewSerdeJSONUnion(union)
		genSerde.SkipImportsAndPackage(true)

		contents = []byte("//union:serde:json\n")
		contents, err = genSerde.Generate()
		if err != nil {
			return shapesContents, fmt.Errorf("mkunion.GenerateUnions: failed to generate json serde for %s: %w", shape.ToGoTypeName(union), err)
		}
		shapesContents.Write(contents)

		pkgMap = generators.MergePkgMaps(pkgMap,
			genSerde.ExtractImports(union),
		)
	}

	contents := bytes.Buffer{}
	contents.WriteString("// Code generated by mkunion. DO NOT EDIT.\n")
	contents.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	contents.WriteString(generators.GenerateImports(pkgMap))
	contents.WriteString(generators.GenerateInitFunc(initFunc))
	_, err = shapesContents.WriteTo(&contents)
	if err != nil {
		return shapesContents, fmt.Errorf("mkunion.GenerateUnions: failed to write shapes contents: %w", err)
	}

	return contents, nil
}

func GenerateSerde(inferred *shape.InferredInfo) (bytes.Buffer, error) {
	shapesContents := bytes.Buffer{}
	shapes := inferred.RetrieveShapesTaggedAs("serde")
	if len(shapes) == 0 {
		return shapesContents, nil
	}

	var err error
	packageName := "main"
	pkgMap := make(generators.PkgMap)
	initFunc := make(generators.InitFuncs, 0, 0)

	for _, x := range shapes {
		packageName = shape.ToGoPkgName(x)
		genSerde := generators.NewSerdeJSONTagged(x)
		genSerde.SkipImportsAndPackage(true)

		contents := "//shape:serde:json\n"
		contents, err = genSerde.Generate()
		if err != nil {
			return shapesContents, fmt.Errorf("mkunion.GenerateSerde: failed to generate json serde for %s: %w", shape.ToGoTypeName(x), err)
		}
		shapesContents.WriteString(contents)

		pkgMap = generators.MergePkgMaps(pkgMap,
			genSerde.ExtractImports(x),
		)
	}

	contents := bytes.Buffer{}
	contents.WriteString("// Code generated by mkunion. DO NOT EDIT.\n")
	contents.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	contents.WriteString(generators.GenerateImports(pkgMap))
	contents.WriteString(generators.GenerateInitFunc(initFunc))

	_, err = shapesContents.WriteTo(&contents)
	if err != nil {
		return shapesContents, fmt.Errorf("mkunion.GenerateSerde: failed to write shapes contents: %w", err)
	}

	return contents, nil
}

func GenerateShape(inferred *shape.InferredInfo) (bytes.Buffer, error) {
	shapesContents := bytes.Buffer{}
	shapes := inferred.RetrieveShapes()
	if len(shapes) == 0 {
		return shapesContents, nil
	}

	packageName := "main"
	pkgMap := make(generators.PkgMap)
	initFunc := make(generators.InitFuncs, 0)

	for _, x := range shapes {
		if shape.TagGetValue(shape.Tags(x), shape.TagShapeName, "") == "-" {
			// skip shape generation for this type
			continue
		}

		packageName = shape.ToGoPkgName(x)
		contents, err := GenerateShapeFollow(x, &pkgMap, &initFunc, inferred)
		if err != nil {
			return shapesContents, fmt.Errorf("mkunion.GenerateShape: failed to generate shape for %s: %w", shape.ToGoTypeName(x), err)
		}
		if contents != nil {
			_, err = contents.WriteTo(&shapesContents)
			if err != nil {
				return shapesContents, fmt.Errorf("mkunion.GenerateShape: failed to write shape for %s: %w", shape.ToGoTypeName(x), err)
			}
		}
	}

	if len(shapesContents.Bytes()) == 0 {
		return shapesContents, nil
	}

	contents := bytes.Buffer{}
	contents.WriteString("// Code generated by mkunion. DO NOT EDIT.\n")
	contents.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	contents.WriteString(generators.GenerateImports(pkgMap))
	contents.WriteString(generators.GenerateInitFunc(initFunc))
	_, err := shapesContents.WriteTo(&contents)
	if err != nil {
		return shapesContents, fmt.Errorf("mkunion.GenerateUnions: failed to write shapes contents: %w", err)
	}

	return contents, nil
}

func GenerateShapeFollow(x shape.Shape, pkgMap *generators.PkgMap, initFunc *[]string, inferred *shape.InferredInfo) (*bytes.Buffer, error) {
	var result *bytes.Buffer
	for _, y := range shape.ExtractRefs(x) {
		// filter types that are not from the same package
		if y.PkgImportName != shape.ToGoPkgImportName(x) {
			log.Debugf("mkunion.GenerateShapeFollow: skipping %s, not from the same package", shape.ToGoTypeName(y))
			continue
		}

		contents, err := GenerateShapeOnce(y, pkgMap, initFunc, inferred)
		if err != nil {
			return nil, fmt.Errorf("mkunion.GenerateShapeFollow: failed to generate shape for %s: %w", shape.ToGoTypeName(y), err)
		}

		if contents == nil {
			continue
		}

		if result == nil {
			result = contents
		} else {
			_, err = contents.WriteTo(result)
			if err != nil {
				return nil, fmt.Errorf("mkunion.GenerateShapeFollow: failed to write shape for %s: %w", shape.ToGoTypeName(y), err)
			}
		}
	}

	return result, nil
}

var _generatedShape = map[string]bool{}

func GenerateShapeOnce(x shape.Shape, pkgMap *generators.PkgMap, initFunc *[]string, inferred *shape.InferredInfo) (*bytes.Buffer, error) {
	key := shape.ToGoTypeName(x, shape.WithPkgImportName())
	if _generatedShape[key] {
		log.Debugf("mkunion.GenerateShapeOnce: shape %s already generated", key)
		return nil, nil
	}

	result := bytes.Buffer{}

	switch x := x.(type) {
	case *shape.RefName:
		y := inferred.RetrieveShapeFromRef(x)
		if y == nil {
			log.Warnf("mkunion.GenerateShapeOnce: failed to lookup shape %s", shape.ToGoTypeName(x, shape.WithPkgImportName()))
			return nil, nil
		}

		switch y := y.(type) {
		case *shape.RefName:
			log.Warnf("mkunion.GenerateShapeOnce: lookup RefName %s", shape.ToGoTypeName(y, shape.WithPkgImportName()))
			return nil, nil
		}

		return GenerateShapeOnce(y, pkgMap, initFunc, inferred)

	case *shape.UnionLike:
		for _, v := range x.Variant {
			key := shape.ToGoTypeName(v, shape.WithPkgImportName())
			_generatedShape[key] = true
		}
	}

	_generatedShape[key] = true

	gen := generators.NewShapeTagged(x)
	gen.SkipImportsAndPackage(true)
	gen.SkipInitFunc(true)

	result.WriteString("//shape:shape\n")
	contents, err := gen.Generate()
	if err != nil {
		return nil, fmt.Errorf("mkunion.GenerateShapeOnce: failed to generate tagged shape for %s: %w", shape.ToGoTypeName(x, shape.WithPkgImportName()), err)
	}
	result.WriteString(contents)

	*pkgMap = generators.MergePkgMaps(*pkgMap,
		gen.ExtractImports(x),
	)

	*initFunc = append(*initFunc, gen.ExtractImportFuncs(x)...)

	return &result, nil
}

func SaveFile(contents bytes.Buffer, sourcePath string, infix string) (string, error) {
	if len(contents.Bytes()) == 0 {
		return "", nil
	}

	sourceName := path.Base(sourcePath)
	baseName := strings.TrimSuffix(sourceName, path.Ext(sourceName))
	fileName := path.Join(
		path.Dir(sourcePath),
		fmt.Sprintf("%s_%s.go", baseName, infix),
	)

	// Format the generated Go code
	formatted, err := format.Source(contents.Bytes())
	if err != nil {
		// Log warning but continue with unformatted code
		log.Warnf("failed to format generated code for %s: %v", fileName, err)
		formatted = contents.Bytes()
	}

	log.Infof("writing %s", fileName)
	err = os.WriteFile(fileName, formatted, 0644)
	if err != nil {
		return fileName, fmt.Errorf("mkunion.SaveFile: failed to write file %s: %w", sourcePath, err)
	}
	return fileName, nil
}

func GenerateTypeRegistry(inferred *shape.IndexedTypeWalker) (bytes.Buffer, error) {
	packageName := inferred.PackageName()

	contents := bytes.Buffer{}
	contents.WriteString("// Code generated by mkunion. DO NOT EDIT.\n")
	contents.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	found := inferred.ExpandedShapes()
	if len(found) == 0 {
		return contents, nil
	}

	sortedKeys := make([]string, 0, len(found))
	for k := range found {
		sortedKeys = append(sortedKeys, k)
	}

	sort.Strings(sortedKeys)

	maps := []generators.PkgMap{
		{
			"shared": "github.com/widmogrod/mkunion/x/shared",
		},
	}
	
	// Add shape package if we have package tags to embed
	pkgTags := inferred.PackageTags()
	if len(pkgTags) > 0 {
		maps = append(maps, generators.PkgMap{
			"shape": "github.com/widmogrod/mkunion/x/shape",
		})
	}
	for _, key := range sortedKeys {
		inst := found[key]
		next := shape.ExtractPkgImportNamesForTypeInitialisation(inst)
		maps = append(maps, next)
	}
	pkgMap := generators.MergePkgMaps(maps...)
	delete(pkgMap, packageName)

	contents.WriteString(generators.GenerateImports(pkgMap))

	contents.WriteString("func init() {\n")
	// generate type registry

	// Embed package tags for runtime access
	if len(pkgTags) > 0 {
		contents.WriteString("\t// Package tags embedded at compile time\n")
		if packageName == "shared" {
			contents.WriteString("\tPackageTagsStore(map[string]interface{}{\n")
		} else {
			contents.WriteString("\tshared.PackageTagsStore(map[string]interface{}{\n")
		}
		
		sortedTagKeys := make([]string, 0, len(pkgTags))
		for k := range pkgTags {
			sortedTagKeys = append(sortedTagKeys, k)
		}
		sort.Strings(sortedTagKeys)
		
		for _, tagKey := range sortedTagKeys {
			tag := pkgTags[tagKey]
			optionsStr := "nil"
			if len(tag.Options) > 0 {
				opts := make([]string, len(tag.Options))
				for i, opt := range tag.Options {
					opts[i] = fmt.Sprintf("%q", opt)
				}
				optionsStr = fmt.Sprintf("[]string{%s}", strings.Join(opts, ", "))
			}
			
			if packageName == "shape" {
				contents.WriteString(fmt.Sprintf("\t\t%q: Tag{Value: %q, Options: %s},\n", 
					tagKey, tag.Value, optionsStr))
			} else {
				contents.WriteString(fmt.Sprintf("\t\t%q: shape.Tag{Value: %q, Options: %s},\n", 
					tagKey, tag.Value, optionsStr))
			}
		}
		contents.WriteString("\t})\n")
	}

	for _, key := range sortedKeys {
		inst := found[key]
		instantiatedTypeName := shape.ToGoTypeName(inst,
			shape.WithInstantiation(),
			shape.WithRootPackage(packageName),
		)
		fullTypeName := shape.ToGoTypeName(inst,
			shape.WithInstantiation(),
			shape.WithPkgImportName(),
		)

		// Register go type
		if packageName == "shared" {
			contents.WriteString(fmt.Sprintf("\tTypeRegistryStore[%s](%q)\n", instantiatedTypeName, fullTypeName))
		} else {
			contents.WriteString(fmt.Sprintf("\tshared.TypeRegistryStore[%s](%q)\n", instantiatedTypeName, fullTypeName))
		}

		// Try to register type JSON marshaller
		if ref, ok := inst.(*shape.RefName); ok {
			some, found := shape.LookupShapeOnDisk(ref)
			if !found {
				continue
			}
			some = shape.IndexWith(some, ref)
			if shape.IsUnion(some) {
				contents.WriteString(fmt.Sprintf("\t%s\n", generators.StrRegisterUnionFuncName(shape.ToGoPkgName(some), some)))
			}
		}
	}

	contents.WriteString("}\n")

	return contents, nil
}

func GenerateMatch(inferred *shape.InferredInfo) (bytes.Buffer, error) {
	result := bytes.Buffer{}

	match := generators.NewMkMatchTaggedNodeVisitor()
	match.FromInferredInfo(inferred)

	specs := match.Specs()
	if len(specs) == 0 {
		return result, nil
	}

	derived := generators.MkMatchGenerator{
		Header:      "// Code generated by mkunion. DO NOT EDIT.",
		PackageName: inferred.PackageName(),
		MatchSpecs:  specs,
	}

	b, err := derived.Generate()

	if err != nil {
		return result, fmt.Errorf("GenerateMatch: failed to generate match: %w", err)
	}

	result.Write(b)
	return result, nil
}

func runGoGenerate(dirs []string) error {
	if len(dirs) == 0 {
		return nil
	}

	// Extract unique directories from the provided paths
	uniqueDirs := make(map[string]bool)
	for _, dir := range dirs {
		uniqueDirs[dir] = true
	}

	// Filter directories that actually contain Go files with generate directives
	dirsToProcess := []string{}
	for dir := range uniqueDirs {
		if hasGoGenerateDirectives(dir) {
			dirsToProcess = append(dirsToProcess, dir)
		}
	}

	if len(dirsToProcess) == 0 {
		log.Debug("No directories with go:generate directives found, skipping go generate")
		return nil
	}

	// Run go generate for each directory with generate directives
	hasErrors := false
	for _, dir := range dirsToProcess {
		log.Infof("Running 'go generate ./...' in %s", dir)

		cmd := exec.Command("go", "generate", "./...")
		cmd.Dir = dir

		// Capture output to filter out non-error messages
		output, err := cmd.CombinedOutput()

		if err != nil {
			// Check if it's a real error or just "no packages" warning
			outputStr := string(output)
			if strings.Contains(outputStr, "matched no packages") {
				log.Debugf("No packages to generate in %s", dir)
				continue
			}

			// Real error - log it as warning
			log.Warnf("'go generate ./...' failed in %s: %v\n%s", dir, err, outputStr)
			hasErrors = true
			continue
		}

		// Log output if any (successful generation)
		if len(output) > 0 {
			log.Debugf("go generate output for %s:\n%s", dir, string(output))
		}
	}

	if hasErrors {
		// Return error silently - we already logged specific warnings
		return fmt.Errorf("go generate failed in one or more directories")
	}

	if len(dirsToProcess) > 0 {
		log.Info("Finished running 'go generate ./...'")
	}
	return nil
}

// hasGoGenerateDirectives checks if a directory contains any Go files with //go:generate directives
func hasGoGenerateDirectives(dir string) bool {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(files) == 0 {
		return false
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Quick check for go:generate directive
		if bytes.Contains(content, []byte("//go:generate")) {
			return true
		}
	}

	// Also check subdirectories (for "./..." pattern)
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".go" {
			content, err := os.ReadFile(path)
			if err == nil && bytes.Contains(content, []byte("//go:generate")) {
				return fmt.Errorf("found") // Use error to stop walking
			}
		}
		return nil
	})

	return err != nil && err.Error() == "found"
}
