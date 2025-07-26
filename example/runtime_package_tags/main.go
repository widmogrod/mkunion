//go:tag version:"2.0.0,stable,production"
//go:tag module:"runtime-tags-example"
//go:tag author:"mkunion development team"
//go:tag license:"MIT"
package main

import (
	"fmt"
	"log"

	"github.com/widmogrod/mkunion/x/shape"
)

//go:tag mkunion:"TaskStatus"
type (
	Pending   struct{ ID string }
	Running   struct{ ID string; Progress float64 }
	Completed struct{ ID string; Result string }
	Failed    struct{ ID string; Error string }
)

func demonstrateSourceFileAccess() {
	fmt.Println("=== Source File Access (requires source files) ===")
	
	// Extract package-level tags from this file
	tags, err := shape.ExtractPackageTagsFromFile("main.go")
	if err != nil {
		log.Printf("Could not read source file: %v", err)
		return
	}

	fmt.Println("Package-level tags found in source:")
	for tagName, tag := range tags {
		fmt.Printf("  %s: %q", tagName, tag.Value)
		if len(tag.Options) > 0 {
			fmt.Printf(" (options: %v)", tag.Options)
		}
		fmt.Println()
	}

	// Using convenience functions with source file tags
	version := shape.GetPackageTagValue(tags, "version", "unknown")
	fmt.Printf("\nSource file version: %s\n", version)
}

func demonstrateRuntimeAccess() {
	fmt.Println("\n=== Runtime Access (works in static binaries) ===")
	
	const pkgImportName = "github.com/widmogrod/mkunion/example/runtime_package_tags"
	
	// Get runtime-embedded tags for our specific package
	tags := shape.GetRuntimePackageTagsForPackage(pkgImportName)
	
	if len(tags) == 0 {
		fmt.Println("No runtime tags found (type registry might be disabled)")
		return
	}
	
	fmt.Println("Runtime-embedded package tags:")
	for tagName, tag := range tags {
		fmt.Printf("  %s: %q", tagName, tag.Value)
		if len(tag.Options) > 0 {
			fmt.Printf(" (options: %v)", tag.Options)
		}
		fmt.Println()
	}

	// Using convenience functions with runtime tags
	version := shape.GetRuntimePackageTagValueForPackage(pkgImportName, "version", "unknown")
	author := shape.GetRuntimePackageTagValueForPackage(pkgImportName, "author", "anonymous")
	license := shape.GetRuntimePackageTagValueForPackage(pkgImportName, "license", "proprietary")
	
	fmt.Printf("\nRuntime metadata:\n")
	fmt.Printf("  Version: %s\n", version)
	fmt.Printf("  Author: %s\n", author)
	fmt.Printf("  License: %s\n", license)

	// Check tag options
	if shape.HasRuntimePackageTagOptionForPackage(pkgImportName, "version", "production") {
		fmt.Println("  ✓ Production-ready release")
	}
	
	// Demonstrate backward compatibility: old functions still work but return namespaced keys
	fmt.Println("\n=== Backward Compatibility ===")
	allTags := shape.GetRuntimePackageTags()
	fmt.Printf("GetRuntimePackageTags() returns %d tags with namespaced keys:\n", len(allTags))
	for namespacedKey := range allTags {
		fmt.Printf("  %s\n", namespacedKey)
	}
	
	if shape.HasRuntimePackageTagOption("version", "stable") {
		fmt.Println("  ✓ Stable version")
	}
}

func main() {
	fmt.Println("Package Tags Runtime Access Example")
	fmt.Println("===================================")
	
	// Demonstrate both approaches
	demonstrateSourceFileAccess()
	demonstrateRuntimeAccess()
	
	fmt.Println("\n=== Comparison ===")
	fmt.Println("Source file access:")
	fmt.Println("  ✓ Works during development")
	fmt.Println("  ✗ Requires source files")
	fmt.Println("  ✗ Fails in deployed binaries")
	
	fmt.Println("\nRuntime access:")
	fmt.Println("  ✓ Works in static binaries")
	fmt.Println("  ✓ No source files required")
	fmt.Println("  ✓ Perfect for deployment metadata")
	fmt.Println("  ✗ Requires type registry generation")
	
	// Example of using status types
	fmt.Println("\n=== Working with Union Types ===")
	
	task := &Running{ID: "task-123", Progress: 0.75}
	fmt.Printf("Current task status: %+v\n", task)
}