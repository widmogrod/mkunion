# x/storage - Go schemaless ORM with union type support
Make working with data types in Go easy and simple. 
Focus on business logic that differentiates your needs, not on serialization and deserialization.


```go
type MyRecord struct {
	Age int
	Name string
}

store := schemaless.NewInMemoryRepository()
//store := schemaless.NewDynamoDBRepository(dynamodb.NewFromConfig(cfg), tableName)
//store := NewOpenSearchRepository(client, indexName)

// Make working with records type-safe
repo := typedful.NewTypedRepository[MyRecord](store)
state, err := repo.Get("1", "user")
assert.ErrorIs(t, err, schemaless.ErrNotFound)

err = repo.UpdateRecords(schemaless.Save(schemaless.Record[MyRecord]{
    ID:   "1",
    Type: "user",
    Data: MyRecord{...},
}))
assert.NoError(t, err)

result, err := repo.FindingRecords(FindingRecords[Record[MyRecord]]{
    Where: predicate.MustWhere(
        "Data.#.Age > :age",
        predicate.ParamBinds{
            ":age": schema.MkInt(20),
        },
    ),
    Sort: []SortField{
        {
            Field:      "Data.#.Name",
            Descending: false,
        },
    },
})
```

## Roadmap
### V0.1.0
- [x] x/storage support DynamoDB, OpenSearch, and InMemory implementation
