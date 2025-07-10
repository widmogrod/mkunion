---
title: Storage Package
---

# x/storage - Schemaless Storage with Union Type Support

The `x/storage` package provides a flexible, type-safe storage abstraction layer for Go applications. It combines the flexibility of schemaless databases with the type safety of Go's generics and mkunion's union types.

## Overview

The storage package offers:
- **Schemaless design** - Store any Go type without predefined schemas
- **Multiple backends** - In-memory, DynamoDB, and OpenSearch implementations
- **Type-safe operations** - Full generics support with compile-time type checking
- **Powerful queries** - Predicate-based queries with support for complex conditions
- **Union type integration** - First-class support for mkunion union types
- **Optimistic concurrency** - Version-based conflict resolution

## Core Concepts

### Repository Interface

The `Repository[T any]` interface provides three core operations:

```go
type Repository[T any] interface {
    // Get retrieves a single record by ID and type
    Get(ctx context.Context, recordID, recordType string) (Record[T], error)
    
    // UpdateRecords performs batch save/delete operations
    UpdateRecords(ctx context.Context, cmd UpdateRecordsRequest[T]) error
    
    // FindingRecords queries records with predicates
    FindingRecords(ctx context.Context, query FindingRecords[T]) (PageResult[T], error)
}
```

### Record Structure

Records are the fundamental unit of storage:

```go
type Record[T any] struct {
    ID      string  // Unique identifier
    Type    string  // Record type/category
    Data    T       // The actual data
    Version int     // For optimistic concurrency
}
```

## Getting Started

### Basic Usage

```go
import (
    "github.com/widmogrod/mkunion/x/storage/schemaless"
    "github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
)

// Define your data type
type User struct {
    Name  string
    Email string
    Age   int
}

// Create an in-memory storage
storage := schemaless.NewInMemoryRepository()

// Wrap with type-safe repository
repo := typedful.NewTypedRepository[User](storage)

// Save a record
err := repo.UpdateRecords(ctx, schemaless.Save(
    schemaless.Record[User]{
        ID:   "user-1",
        Type: "user",
        Data: User{
            Name:  "Alice",
            Email: "alice@example.com",
            Age:   30,
        },
    },
))

// Retrieve a record
record, err := repo.Get(ctx, "user-1", "user")
fmt.Printf("User: %+v\n", record.Data)
```

### Querying with Predicates

The storage package provides a powerful predicate query system:

```go
// Find all users older than 25
results, err := repo.FindingRecords(ctx, schemaless.FindingRecords[User]{
    RecordType: "user",
    Where: predicate.MustWhere(
        "Data.Age > :age",
        predicate.ParamBinds{
            ":age": schema.MkInt(25),
        },
    ),
    Sort: []schemaless.SortField{
        {Field: "Data.Name", Descending: false},
    },
})

// Iterate through results
for _, record := range results.Items {
    fmt.Printf("User: %s (age: %d)\n", record.Data.Name, record.Data.Age)
}
```

## Storage Backends

### In-Memory Storage

Thread-safe, fast storage suitable for testing and caching:

```go
storage := schemaless.NewInMemoryRepository()
```

Features:
- Mutex-protected for concurrent access
- Maintains an append log of all operations
- Perfect for unit tests and development

### DynamoDB Storage

Scalable, managed NoSQL storage on AWS:

```go
storage, err := schemaless.NewDynamoDBRepository(dynamoDBClient, schemaless.DynamoDBConfig{
    TableName:      "my-table",
    PartitionKey:   "PK",
    SortKey:        "SK",
    GSI:           "GSI1",
    GSIPartitionKey: "GSI1PK",
    GSISortKey:     "GSI1SK",
})
```

Features:
- Transaction support for batch operations
- Automatic retry with exponential backoff
- Efficient querying with GSI support

### OpenSearch Storage

Full-text search capabilities:

```go
storage, err := schemaless.NewOpenSearchRepository(opensearchClient, schemaless.OpenSearchConfig{
    Index: "my-index",
})
```

Features:
- Full-text search on all fields
- Complex aggregations
- Near real-time indexing

## Advanced Features

### Union Type Support

The storage package seamlessly handles mkunion union types:

```go
//go:tag mkunion:"Event"
type (
    UserCreated struct {
        UserID string
        Name   string
    }
    UserUpdated struct {
        UserID string
        Changes map[string]any
    }
)

// Store union types directly
repo := typedful.NewTypedRepository[Event](storage)
err := repo.UpdateRecords(ctx, schemaless.Save(
    schemaless.Record[Event]{
        ID:   "evt-1",
        Type: "event",
        Data: &UserCreated{UserID: "u1", Name: "Alice"},
    },
))
```

### Batch Operations

Perform multiple operations atomically:

```go
err := repo.UpdateRecords(ctx, schemaless.UpdateRecordsRequest[User]{
    Saving: []schemaless.Record[User]{
        {ID: "u1", Type: "user", Data: user1},
        {ID: "u2", Type: "user", Data: user2},
    },
    Deleting: []schemaless.Record[User]{
        {ID: "u3", Type: "user", Version: 5},
    },
    UpdatePolicy: schemaless.PolicyIfServerNotChanged,
})
```

### Pagination

Handle large result sets efficiently:

```go
// First page
page1, _ := repo.FindingRecords(ctx, schemaless.FindingRecords[User]{
    RecordType: "user",
    Limit:      10,
})

// Next page using cursor
page2, _ := repo.FindingRecords(ctx, schemaless.FindingRecords[User]{
    RecordType: "user",
    Limit:      10,
    After:      page1.Next,
})
```

### Complex Queries

Build sophisticated queries with the predicate system:

```go
// Complex query with AND/OR conditions
where := predicate.MustWhere(
    "(Data.Status = :active AND Data.Age >= :minAge) OR Data.Role = :admin",
    predicate.ParamBinds{
        ":active": schema.MkString("active"),
        ":minAge": schema.MkInt(18),
        ":admin":  schema.MkString("admin"),
    },
)

// Query with sorting on multiple fields
results, _ := repo.FindingRecords(ctx, schemaless.FindingRecords[User]{
    RecordType: "user",
    Where:      where,
    Sort: []schemaless.SortField{
        {Field: "Data.Role", Descending: false},
        {Field: "Data.CreatedAt", Descending: true},
    },
})
```

## Projection System

The storage package includes a powerful projection system for event processing:

```go
// Create a projection from events to aggregated state
projection := projection.New[Event, UserStats](
    eventStream,
    handler.NewAggregator[Event, UserStats](),
)

// The projection automatically:
// - Processes events in order
// - Maintains aggregated state
// - Handles failures and recovery
// - Supports windowing for time-based aggregations
```

## Best Practices

### 1. Use Type-Safe Wrappers

Always wrap the generic repository with `TypedRepository`:

```go
// Good
typedRepo := typedful.NewTypedRepository[MyType](storage)

// Avoid using raw schema-based repository directly
```

### 2. Handle Versioning

Use optimistic concurrency control for updates:

```go
// Get current version
record, _ := repo.Get(ctx, id, recordType)

// Update with version check
record.Data.Name = "Updated Name"
err := repo.UpdateRecords(ctx, schemaless.Save(record))
// Will fail if another process updated the record
```

### 3. Design Record Types

Group related data into appropriate record types:

```go
// Good: Separate types for different entities
type UserRecord struct { /* user fields */ }
type OrderRecord struct { /* order fields */ }

// Store with meaningful type names
repo.Save(Record[any]{Type: "user", Data: userData})
repo.Save(Record[any]{Type: "order", Data: orderData})
```

### 4. Query Performance

- Use indexes effectively (especially with DynamoDB GSI)
- Limit result sets with pagination
- Design record types to minimize cross-type queries
- Use appropriate backends for your query patterns

## Migration Guide

### From SQL Databases

```go
// SQL approach
rows, _ := db.Query("SELECT * FROM users WHERE age > ?", 25)

// Storage package approach
results, _ := repo.FindingRecords(ctx, schemaless.FindingRecords[User]{
    RecordType: "user",
    Where: predicate.MustWhere("Data.Age > :age", 
        predicate.ParamBinds{":age": schema.MkInt(25)}),
})
```

### From Document Stores

```go
// MongoDB approach
cursor, _ := collection.Find(bson.M{"age": bson.M{"$gt": 25}})

// Storage package approach
results, _ := repo.FindingRecords(ctx, schemaless.FindingRecords[User]{
    RecordType: "user",
    Where: predicate.MustWhere("Data.Age > :age",
        predicate.ParamBinds{":age": schema.MkInt(25)}),
})
```

## Error Handling

The storage package follows Go's standard error handling conventions:

```go
record, err := repo.Get(ctx, id, recordType)
if err != nil {
    if errors.Is(err, schemaless.ErrNotFound) {
        // Record doesn't exist
    } else if errors.Is(err, schemaless.ErrVersionConflict) {
        // Concurrent modification detected
    } else {
        // Other error
    }
}
```

## Performance Considerations

1. **In-Memory**: O(n) queries, best for small datasets (<100k records)
2. **DynamoDB**: Optimized for key-based access, use GSI for queries
3. **OpenSearch**: Best for full-text search and complex aggregations

## Integration with Other x/ Packages

The storage package integrates seamlessly with:
- **x/machine**: Store state machine states
- **x/workflow**: Persist workflow state
- **x/projection**: Source for event projections
- **x/shape**: Automatic schema inference

## Troubleshooting

Common issues and solutions:

1. **Version conflicts**: Ensure you're reading the latest version before updates
2. **Query performance**: Add appropriate indexes for your query patterns
3. **Type registration**: Register union types for JSON marshalling
4. **Memory usage**: Use pagination for large result sets

## Further Reading

- [Predicate Query Language](../examples/predicate_queries.md)
- [Storage Backend Comparison](../development/storage_backends.md)
- [Event Sourcing with Projections](./projection.md)
- [State Machine Persistence](./machine.md#persistence)