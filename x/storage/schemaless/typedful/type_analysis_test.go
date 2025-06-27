package typedful

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"testing"
)

// TestTypeAnalysis analyzes the type conversion issue
func TestTypeAnalysis(t *testing.T) {
	// Create a record where Data is already schema.Schema
	record := schemaless.Record[schema.Schema]{
		ID:   "test1",
		Type: "test",
		Data: schema.MkMap(schema.MkField("Count", schema.MkInt(42))),
	}

	fmt.Println("=== Original record ===")
	fmt.Printf("Type of record: %T\n", record)
	fmt.Printf("Type of record.Data: %T\n", record.Data)
	fmt.Printf("Value of record.Data: %v\n", record.Data)

	// What happens when we convert it with FromGo?
	schemaRepresentation := schema.FromGo(record)
	fmt.Println("\n=== After FromGo ===")
	fmt.Printf("Type: %T\n", schemaRepresentation)

	// Let's inspect the structure
	if m, ok := schemaRepresentation.(*schema.Map); ok {
		fmt.Println("Top level is a Map with fields:")
		for k, v := range *m {
			fmt.Printf("  %s: %T = %v\n", k, v, v)
		}

		// What's in the Data field?
		if dataField, ok := (*m)["Data"]; ok {
			fmt.Printf("\nData field type: %T\n", dataField)

			// Is it a Map like we expect?
			if dataMap, ok := dataField.(*schema.Map); ok {
				fmt.Println("Data field is a Map with:")
				for k, v := range *dataMap {
					fmt.Printf("  %s: %T = %v\n", k, v, v)
				}
			}
		}
	}

	// Now let's see what GetSchemaLocation returns for different paths
	fmt.Println("\n=== GetSchema results ===")
	paths := []string{"Data", "Data.Count"}
	for _, path := range paths {
		result, found := schema.GetSchema(schemaRepresentation, path)
		fmt.Printf("Path '%s': found=%v, type=%T, value=%v\n", path, found, result, result)
	}
}
