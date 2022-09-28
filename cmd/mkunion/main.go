package main

import (
	"bytes"
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
		Name:                   "mkunion",
		Description:            "UnionVisitorGenerator union type and visitor pattern gor golang",
		EnableBashCompletion:   true,
		DefaultCommand:         "golang",
		UseShortOptionHandling: true,
		Commands: []*cli.Command{
			{
				Name: "golang",
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
					&cli.StringFlag{
						Name:     "output",
						Aliases:  []string{"o"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "package",
						Aliases:  []string{"p"},
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					cwd, _ := syscall.Getwd()
					file := path.Join(cwd, os.Getenv("GOFILE"))
					inferred, err := mkunion.InferFromFile(file)
					if err != nil {
						return err
					}
					u := mkunion.UnionVisitorGenerator{
						Types: strings.Split(c.String("types"), ","),
						Name:  c.String("name"),
						//PackageName: c.String("package"),
						PackageName: inferred.PackageName,
					}

					t := mkunion.TraverseGenerator{
						Name: u.Name,
						//PackageName: u.PackageName,
						PackageName: inferred.PackageName,
						Types:       u.Types,
						Branches:    inferred.ForVariantType(u.Name, u.Types),
						NoHeader:    true,
					}

					unionVisitor, err := u.Generate()
					if err != nil {
						return err
					}

					traverse, err := t.Generate()
					if err != nil {
						return err
					}

					bb := bytes.Buffer{}
					bb.Write(unionVisitor)
					bb.Write(traverse)

					err = ioutil.WriteFile(c.String("output"), bb.Bytes(), 0644)
					if err != nil {
						return err
					}

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
