package main

import (
	"flag"
	"github.com/widmogrod/mkunion"
	"io/ioutil"
	"strings"
)

var output = flag.String("output", "-", "output to *.dpr file")
var types = flag.String("types", "", "output to *.dpr file")
var name = flag.String("name", "", "output to *.dpr file")
var packageName = flag.String("packageName", "main", "go package name")

func main() {
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

	err = ioutil.WriteFile(*output+".go", result, 0644)
	if err != nil {
		panic(err)
	}
}
