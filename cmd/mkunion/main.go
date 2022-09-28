package main

import (
	"context"
	"github.com/urfave/cli/v2"
	"github.com/widmogrod/mkunion"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	var app *cli.App
	app = &cli.App{
		Name:                   "mkunion",
		Description:            "Generate union type and visitor pattern gor golang",
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
					g := mkunion.Generate{
						Types:       strings.Split(c.String("types"), ","),
						Name:        c.String("name"),
						PackageName: c.String("package"),
					}

					result, err := g.Generate()
					if err != nil {
						panic(err)
					}

					err = ioutil.WriteFile(c.String("output"), result, 0644)
					if err != nil {
						panic(err)
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
