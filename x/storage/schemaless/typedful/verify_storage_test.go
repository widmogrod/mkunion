package typedful

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"testing"
)

// TestVerifyStorageTypes verifies the type mismatch in storage
func TestVerifyStorageTypes(t *testing.T) {
	storage := schemaless.NewInMemoryRepository[schema.Schema]()

	// Simulate what happens in the aggregator

	// 1. User record - converted from Go type
	userRecord := schemaless.Record[User]{
		ID:   "user1",
		Type: "user",
		Data: User{Name: "John", Age: 25},
	}
	userSchema := schema.FromGo(userRecord.Data)
	storage.UpdateRecords(schemaless.Save(schemaless.Record[schema.Schema]{
		ID:   userRecord.ID,
		Type: userRecord.Type,
		Data: userSchema,
	}))

	// 2. Index record - already schema.Schema
	indexRecord := schemaless.Record[schema.Schema]{
		ID:   "byAge:20-30",
		Type: "byAge",
		Data: schema.MkMap(schema.MkField("Count", schema.MkInt(1))),
	}
	storage.UpdateRecords(schemaless.Save(indexRecord))

	// Now retrieve and inspect both
	fmt.Println("=== Comparing stored records ===")

	userStored, _ := storage.Get("user1", "user")
	fmt.Printf("\nUser record - Data type: %T\n", userStored.Data)
	if m, ok := userStored.Data.(*schema.Map); ok {
		fmt.Println("User Data fields:")
		for k, v := range *m {
			fmt.Printf("  %s: %T\n", k, v)
		}
	}

	indexStored, _ := storage.Get("byAge:20-30", "byAge")
	fmt.Printf("\nIndex record - Data type: %T\n", indexStored.Data)
	if m, ok := indexStored.Data.(*schema.Map); ok {
		fmt.Println("Index Data fields:")
		for k, v := range *m {
			fmt.Printf("  %s: %T\n", k, v)
		}
	}

	// Test schema.Get on both
	fmt.Println("\n=== Testing schema.Get ===")
	userConverted := schema.FromGo(userStored)
	indexConverted := schema.FromGo(indexStored)

	// Try to get nested fields
	userData, found := schema.GetSchema(userConverted, "Data.Name")
	fmt.Printf("User Data.Name: found=%v, value=%v\n", found, userData)

	indexData, found := schema.GetSchema(indexConverted, "Data.Count")
	fmt.Printf("Index Data.Count: found=%v, value=%v\n", found, indexData)
}
