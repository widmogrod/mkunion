package main

import (
	"context"
	"github.com/urfave/cli/v2"
	"github.com/widmogrod/mkunion"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	var app *cli.App
	app = &cli.App{
		Name:                   mkunion.Program,
		Description:            "VisitorGenerator union type and visitor pattern gor golang",
		EnableBashCompletion:   true,
		DefaultCommand:         "golang",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n", "variant"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "types",
				Aliases:  []string{"t"},
				Required: false,
			},
			&cli.StringFlag{
				Name:     "skip-extension",
				Aliases:  []string{"skip-ext"},
				Required: false,
			},
		},
		Action: func(c *cli.Context) error {
			cwd, _ := syscall.Getwd()
			sourceName := path.Base(os.Getenv("GOFILE"))
			sourcePath := path.Join(cwd, sourceName)

			baseName := strings.TrimSuffix(sourceName, path.Ext(sourceName))

			// file name without extension
			inferred, err := mkunion.InferFromFile(sourcePath)
			if err != nil {
				return err
			}

			unionName := c.String("name")
			var types []string
			if c.String("types") != "" {
				types = strings.Split(c.String("types"), ",")
			} else {
				types = inferred.PossibleVariantsTypes(unionName)
			}

			visitor := mkunion.VisitorGenerator{
				Header:      mkunion.Header,
				Name:        unionName,
				Types:       types,
				PackageName: inferred.PackageName,
			}

			depthFirstGenerator := mkunion.ReducerDepthFirstGenerator{
				Header:      mkunion.Header,
				Name:        visitor.Name,
				Types:       visitor.Types,
				PackageName: inferred.PackageName,
				Branches:    inferred.ForVariantType(visitor.Name, visitor.Types),
			}

			breadthFirstGenerator := mkunion.ReducerBreadthFirstGenerator{
				Header:      mkunion.Header,
				Name:        visitor.Name,
				Types:       visitor.Types,
				PackageName: inferred.PackageName,
				Branches:    inferred.ForVariantType(visitor.Name, visitor.Types),
			}

			defaultReduction := mkunion.ReducerDefaultReductionGenerator{
				Header:      mkunion.Header,
				Name:        visitor.Name,
				Types:       visitor.Types,
				PackageName: inferred.PackageName,
			}

			defaultVisitor := mkunion.VisitorDefaultGenerator{
				Header:      mkunion.Header,
				Name:        visitor.Name,
				Types:       visitor.Types,
				PackageName: inferred.PackageName,
			}

			schema := mkunion.DeSerJsonGenerator{
				Header:      mkunion.Header,
				Name:        visitor.Name,
				Types:       visitor.Types,
				PackageName: inferred.PackageName,
			}

			generators := map[string]mkunion.Generator{
				"visitor":         &visitor,
				"reducer_dfs":     &depthFirstGenerator,
				"reducer_bfs":     &breadthFirstGenerator,
				"default_reducer": &defaultReduction,
				"default_visitor": &defaultVisitor,
				"schema":          &schema,
			}

			skipExtension := strings.Split(c.String("skip-extension"), ",")
			for _, name := range skipExtension {
				delete(generators, name)
			}

			for name, g := range generators {
				b, err := g.Generate()
				if err != nil {
					return err
				}
				err = os.WriteFile(path.Join(cwd,
					baseName+"_"+mkunion.Program+"_"+strings.ToLower(visitor.Name)+"_"+name+".go"), b, 0644)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
