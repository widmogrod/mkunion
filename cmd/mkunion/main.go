package main

import (
	"context"
	"github.com/urfave/cli/v2"
	"github.com/widmogrod/mkunion"
	"io/ioutil"
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
				Required: true,
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
			visitor := mkunion.VisitorGenerator{
				Name:        c.String("name"),
				Types:       strings.Split(c.String("types"), ","),
				PackageName: inferred.PackageName,
			}

			reducer := mkunion.ReducerGenerator{
				Name:        visitor.Name,
				Types:       visitor.Types,
				PackageName: inferred.PackageName,
				Branches:    inferred.ForVariantType(visitor.Name, visitor.Types),
			}

			defaultVisitor := mkunion.VisitorDefaultGenerator{
				Name:        visitor.Name,
				Types:       visitor.Types,
				PackageName: inferred.PackageName,
			}

			generators := []struct {
				gen  mkunion.Generator
				name string
			}{
				{gen: &visitor, name: "visitor"},
				{gen: &reducer, name: "reducer"},
				{gen: &defaultVisitor, name: "default_visitor"},
			}
			for _, g := range generators {
				b, err := g.gen.Generate()
				if err != nil {
					return err
				}
				err = ioutil.WriteFile(path.Join(cwd,
					baseName+"_"+mkunion.Program+"_"+g.name+".go"), b, 0644)
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
