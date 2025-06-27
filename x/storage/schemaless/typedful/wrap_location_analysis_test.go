package typedful

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"testing"
)

// TestWrapLocationAnalysis analyzes how WrapLocationStr works
func TestWrapLocationAnalysis(t *testing.T) {
	// Create TypedLocation for User records
	encodedAs, _ := shape.LookupShapeReflectAndIndex[schemaless.Record[schema.Schema]]()

	// Test 1: TypedLocation for Record[User]
	userLoc, _ := schema.NewTypedLocationWithEncoded[schemaless.Record[User]](encodedAs)

	// Test 2: TypedLocation for Record[UsersCountByAge]
	countLoc, _ := schema.NewTypedLocationWithEncoded[schemaless.Record[UsersCountByAge]](encodedAs)

	fmt.Println("=== WrapLocationStr Analysis ===")

	// Test wrapping for User record paths
	fmt.Println("\nUser record paths:")
	userPaths := []string{"Data.Name", "Data.Age", "ID", "Type"}
	for _, path := range userPaths {
		wrapped, err := userLoc.WrapLocationStr(path)
		fmt.Printf("  %s -> %s (err: %v)\n", path, wrapped, err)
	}

	// Test wrapping for UsersCountByAge record paths
	fmt.Println("\nUsersCountByAge record paths:")
	countPaths := []string{"Data.Count", "ID", "Type"}
	for _, path := range countPaths {
		wrapped, err := countLoc.WrapLocationStr(path)
		fmt.Printf("  %s -> %s (err: %v)\n", path, wrapped, err)
	}

	// Now test if these wrapped paths work with actual data
	fmt.Println("\n=== Testing wrapped paths with actual data ===")

	// Create a User record and convert to schema
	userRecord := schemaless.Record[User]{
		ID:   "user1",
		Type: "user",
		Data: User{Name: "John", Age: 25},
	}
	userSchema := schema.FromGo(userRecord.Data)
	userRecordSchema := schemaless.Record[schema.Schema]{
		ID:   userRecord.ID,
		Type: userRecord.Type,
		Data: userSchema,
	}
	userRecordConverted := schema.FromGo(userRecordSchema)

	// Create a UsersCountByAge record and convert to schema
	countRecord := schemaless.Record[UsersCountByAge]{
		ID:   "byAge:20-30",
		Type: "byAge",
		Data: UsersCountByAge{Count: 1},
	}
	countSchema := schema.FromGo(countRecord.Data)
	countRecordSchema := schemaless.Record[schema.Schema]{
		ID:   countRecord.ID,
		Type: countRecord.Type,
		Data: countSchema,
	}
	countRecordConverted := schema.FromGo(countRecordSchema)

	// Test user record with wrapped path
	wrappedUserPath, _ := userLoc.WrapLocationStr("Data.Name")
	userVal, userFound := schema.GetSchema(userRecordConverted, wrappedUserPath)
	fmt.Printf("\nUser record - GetSchema('%s'): found=%v, value=%v\n", wrappedUserPath, userFound, userVal)

	// Test count record with wrapped path
	wrappedCountPath, _ := countLoc.WrapLocationStr("Data.Count")
	countVal, countFound := schema.GetSchema(countRecordConverted, wrappedCountPath)
	fmt.Printf("Count record - GetSchema('%s'): found=%v, value=%v\n", wrappedCountPath, countFound, countVal)

	// What about the original path without wrapping?
	origVal, origFound := schema.GetSchema(countRecordConverted, "Data.Count")
	fmt.Printf("Count record - GetSchema('Data.Count'): found=%v, value=%v\n", origFound, origVal)
}
