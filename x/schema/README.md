# Golang recursive schema
Library allows to write code that work with any type of schemas.
Regardless if those are JSON, XML, YAML, or golang structs.

Most benefits
- Union types can be deserialized into interface field

## How to convert between json <-> go
```go
data := `{"name": "John", "cars": [{"name":"Ford"}]}`
schema := schema.FromJSON(data)
nativego, err := schema.ToGo(schema)

expected := map[string]any{
    "name": "John",
    "cars": []any{
        map[string]any{
            "name": "Ford",
        },
    },
}
assert.Equal(t, expected, nativego)
```

## How to convert schema into named golang struct?
This example shows how to convert only part of schema to golang struct.
List of cars will have type `Car` when parent `Person` object will be `map[string]any`.
```go
type Car struct {
    Name string
}
nativego := schema.MustToGo(schema, WithOnlyTheseRules(
	WhenPath([]string{"cars", "[*]"}, UseStruct(Car{}))))

expected := map[string]any{
    "name": "John",
    "cars": []any{
        Car{
            Name: "Ford",
        },
    },
}
assert.Equal(t, expected, nativego)
```

## Roadmap
### V0.1.0
- [x] Json <-> Schema <-> Go (with structs mapping)
- [x] Write test with wrong type conversions
- [x] Value are split into Number(Int, Float), String, Bool, and Null
- [x] Default schema registry + mkunion make union serialization/deserialization work transperently
- [x] Support pointers *string, etc.
- [x] Support DynamoDB (FromDynamoDB, ToDynamoDB)
- [x] Support for pointer to types like *string, *int, etc.
- [x] Support for relative paths like `WhenPath([]string{"*", "ListOfCars", "Car"}, UseStruct(Car{}))`. 
      Absolute paths are without `*` at the beginning.
 
### V0.2.x
- [x] Support options for `ToGo` & `FromGo` like `WithOnlyTheseRules`, `WithExtraRules`, `WithDefaultMapDef`, etc. 
      Gives better control on how schema is converted to golang.
      It's especially important from security reasons, whey you want to allow rules only whitelisted rules, for user generated json input.
- [x] Schema support interface for custom type-setters, that don't require reflection, and mkunion can leverage them. Use `UseTypeDef` eg. `WhenPath([]string{}, UseTypeDef(&someTypeDef{})),`
- [x] Support for how union names should be expressed in schema `WithUnionFormatter(func(t reflect.Type) string)`

### V0.3.x
- [x] schema.Compare method to compare two schemas

### V0.4.x
- [x] Support for Binary type
- [x] Add missing function for MkBinary, MkFloat, MkNone

### V0.5.x
- [x] `schema.UnwrapDynamoDB` takes DynamoDB specific nesting and removes it.
- [x] Eliminate data races in `*UnionVariants[A] -> MapDefFor` & `UseUnionFormatter` data race
- [x] Introduce `ToGoG[T any]` function, that makes ToGo with type assertion and tries to convert to T

### V0.6.x
- [ ] Rename `schema.As` to `schema.AsDefault` and make `schema.As` as variant that returns false, if type is not supported
- [ ] `schema.ToGo` can deduce nested types, for fields in struct that have type information
- [ ] Support serialization of schema.Schema to schema.Schema

### V0.7.x
- [ ] Support json tags in golang to map field names to schema
- [ ] Add cata, ana, and hylo morphisms
- [ ] Open `goConfigFunc` to allow customizing how golang types are converted to schema, passed from external code.
