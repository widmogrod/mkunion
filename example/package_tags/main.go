//go:tag mkunion:",no-type-registry"
//go:tag version:"1.0.0,stable,production"
//go:tag module:"package-tags-example"
//go:tag author:"mkunion"
package main

import (
	"fmt"
	"log"

	"github.com/widmogrod/mkunion/x/shape"
)

//go:tag mkunion:"Status"
type (
	Success struct{ Message string }
	Error   struct{ Code int; Message string }
)

func main() {
	// Extract package-level tags from this file
	tags, err := shape.ExtractPackageTagsFromFile("main.go")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Package-level tags found:")
	for tagName, tag := range tags {
		fmt.Printf("  %s: %q", tagName, tag.Value)
		if len(tag.Options) > 0 {
			fmt.Printf(" (options: %v)", tag.Options)
		}
		fmt.Println()
	}

	// Using convenience functions
	version := shape.GetPackageTagValue(tags, "version", "unknown")
	fmt.Printf("\nPackage version: %s\n", version)

	if shape.HasPackageTagOption(tags, "mkunion", "no-type-registry") {
		fmt.Println("Type registry is disabled for this package")
	}

	if shape.HasPackageTagOption(tags, "version", "production") {
		fmt.Println("This is a production-ready package")
	}

	// Example with union types
	var status Status = &Success{Message: "Operation completed successfully"}
	
	result := MatchStatusR1(
		status,
		func(s *Success) string {
			return fmt.Sprintf("✓ %s", s.Message)
		},
		func(e *Error) string {
			return fmt.Sprintf("✗ Error %d: %s", e.Code, e.Message)
		},
	)

	fmt.Printf("\nStatus: %s\n", result)
}