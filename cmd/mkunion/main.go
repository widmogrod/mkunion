package main

import (
	"flag"
	"github.com/widmogrod/mkunion"
	"io/ioutil"
	"strings"
)

var output = flag.String("output", "-", "Output file for generated code")
var types = flag.String("types", "", "Comma separated list of golang types to generate union for")
var name = flag.String("name", "", "Name of the union type")
var packageName = flag.String("package", "main", "go package name")

func main() {
	flag.Usage()
	flag.Parse()

	g := mkunion.Generate{
		Types:       strings.Split(*types, ","),
		Name:        *name,
		PackageName: *packageName,
	}

	result, err := g.Generate()
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(*output, result, 0644)
	if err != nil {
		panic(err)
	}
}
