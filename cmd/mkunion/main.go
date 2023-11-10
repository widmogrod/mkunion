package main

import (
	"bytes"
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/widmogrod/mkunion"
	"github.com/widmogrod/mkunion/x/shape"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// set log level to error
	log.SetOutput(os.Stderr)
	log.SetLevel(log.ErrorLevel)

	var app *cli.App
	app = &cli.App{
		Name:                   mkunion.Program,
		Description:            "VisitorGenerator union type and visitor pattern gor golang",
		EnableBashCompletion:   true,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "variants",
				Aliases:  []string{"var"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "skip-extension",
				Aliases:  []string{"skip-ext"},
				Value:    "reducer_bfs,reducer_dfs,default_visitor,default_reducer",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "include-extension",
				Aliases:  []string{"inc-ext"},
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:      "input-go-file",
				Aliases:   []string{"i", "input"},
				Usage:     `When not provided, it will try to use GOFILE environment variable, used when combined with //go:generate mkunion -name=MyUnionType`,
				TakesFile: true,
			},
			&cli.BoolFlag{
				Name:     "verbose",
				Aliases:  []string{"v"},
				Required: false,
				Value:    false,
			},
			&cli.BoolFlag{
				Name:     "compact-to-one-file",
				Aliases:  []string{"compact"},
				Required: false,
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

			for _, sourcePath := range sourcePaths {
				cwd := path.Dir(sourcePath)
				sourceName := path.Base(sourcePath)
				baseName := strings.TrimSuffix(sourceName, path.Ext(sourceName))

				// file name without extension
				inferred, err := mkunion.InferFromFile(sourcePath)
				if err != nil {
					return err
				}

				unionNamse := c.StringSlice("name")
				if len(unionNamse) == 0 {
					unionNamse = inferred.PossibleUnionTypes()
				}

				for _, unionName := range unionNamse {
					var types []string
					if len(unionNamse) == 1 && c.String("variants") != "" {
						types = strings.Split(c.String("variants"), ",")
					} else {
						types = inferred.PossibleVariantsTypes(unionName)
					}

					options := []mkunion.GenerateOption{
						mkunion.WithPackageName(inferred.PackageName),
					}
					if c.Bool("compact-to-one-file") {
						options = append(options, mkunion.WithBufferedImports())
					}

					helper := mkunion.NewHelper(options...)
					visitor := mkunion.NewVisitorGenerator(unionName, types, helper)
					schema := mkunion.NewSchemaGenerator(visitor.Name, visitor.Types, helper)
					depthFirstGenerator := mkunion.NewReducerDepthFirstGenerator(
						visitor.Name,
						visitor.Types,
						inferred.ForVariantType(visitor.Name, visitor.Types),
						helper,
					)

					breadthFirstGenerator := mkunion.ReducerBreadthFirstGenerator{
						Header:      mkunion.Header,
						Name:        visitor.Name,
						Types:       visitor.Types,
						PackageName: inferred.PackageName,
						Branches:    inferred.ForVariantType(visitor.Name, visitor.Types),
					}

					defaultReduction := mkunion.NewReducerDefaultReductionGenerator(
						visitor.Name,
						visitor.Types,
						helper,
					)

					defaultVisitor := mkunion.NewVisitorDefaultGenerator(visitor.Name, visitor.Types, helper)

					generators := map[string]mkunion.Generator{
						"visitor":         visitor,
						"reducer_dfs":     depthFirstGenerator,
						"reducer_bfs":     &breadthFirstGenerator,
						"default_reducer": defaultReduction,
						"default_visitor": defaultVisitor,
						"schema":          schema,
					}

					skipExtension := strings.Split(c.String("skip-extension"), ",")
					includeExtension := strings.Split(c.String("include-extension"), ",")
					if len(includeExtension) > 0 {
						for _, includeName := range includeExtension {
							for i, skipName := range skipExtension {
								if skipName == includeName {
									log.Infof("include extension, that was skipName %s", includeName)
									skipExtension = append(skipExtension[:i], skipExtension[i+1:]...)
								}
							}
						}
					}

					for _, name := range skipExtension {
						log.Infof("skip extension %s", name)
						delete(generators, name)
					}

					if c.Bool("compact-to-one-file") {
						body := bytes.Buffer{}
						for _, g := range generators {
							b, err := g.Generate()
							if err != nil {
								return err
							}
							body.Write(b)
						}

						header := bytes.Buffer{}
						header.WriteString(helper.RenderBufferedHeader())
						header.WriteString(helper.RenderBufferedImport())
						log.Infof(helper.RenderBufferedImport())

						fileName := baseName + "_" + mkunion.Program + "_" + strings.ToLower(visitor.Name) + ".go"
						log.Infof("writing %s", fileName)

						header.Write(body.Bytes())

						err = os.WriteFile(path.Join(cwd, fileName), header.Bytes(), 0644)
						if err != nil {
							return err
						}
					} else {
						for name, g := range generators {
							b, err := g.Generate()
							if err != nil {
								return err
							}

							fileName := baseName + "_" + mkunion.Program + "_" + strings.ToLower(visitor.Name) + "_" + name + ".go"
							log.Infof("writing %s", fileName)

							err = os.WriteFile(path.Join(cwd, fileName), b, 0644)
							if err != nil {
								return err
							}
						}
					}
				}
			}

			return nil
		},
		Commands: []*cli.Command{
			{
				Name: "match",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					cwd, _ := syscall.Getwd()
					sourceName := path.Base(os.Getenv("GOFILE"))
					sourcePath := path.Join(cwd, sourceName)

					baseName := strings.TrimSuffix(sourceName, path.Ext(sourceName))

					// file name without extension
					inferred, err := mkunion.InferDeriveFuncMatchFromFile(sourcePath)
					if err != nil {
						return err
					}

					specName := c.String("name")
					spec, err := inferred.MatchSpec(specName)
					if err != nil {
						return err
					}

					derived := mkunion.DeriveFuncMatchGenerator{
						Header:      "// Code generated by mkunion. DO NOT EDIT.",
						PackageName: inferred.PackageName,
						MatchSpec:   *spec,
					}

					b, err := derived.Generate()
					if err != nil {
						return err
					}
					err = os.WriteFile(path.Join(
						cwd,
						baseName+"_match_"+strings.ToLower(derived.MatchSpec.Name)+".go"), b, 0644)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name: "shape-export",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "type",
						DefaultText: "typescript",
					},
					&cli.StringFlag{
						Name:    "output-dir",
						Aliases: []string{"o", "output"},
					},
					&cli.StringSliceFlag{
						Name:      "input-go-file",
						Aliases:   []string{"i", "input"},
						Usage:     `When not provided, it will try to use GOFILE environment variable, used when combined with //go:generate mkunion -name=MyUnionType`,
						TakesFile: true,
					},
				},
				Action: func(c *cli.Context) error {
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
						inferred, err := mkunion.InferFromFile(sourcePath)
						if err != nil {
							return err
						}

						for _, union := range inferred.RetrieveUnions() {
							tsr.AddUnion(union)
						}

						for _, structLike := range inferred.RetrieveStruct() {
							tsr.AddStruct(structLike)
						}
					}

					return tsr.WriteToDir(c.String("output-dir"))
				},
			},
		},
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
