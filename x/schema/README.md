# x/schema - Go recursive schema
The library allows you to write code that works with any type of schema,
regardless if those are JSON, XML, YAML, or Go structs.

Key benefits include:
- Union types can be deserialized into an interface field.

## How to convert between JSON <-> Go
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

## How to convert schema into a named Go struct?
This example shows how to convert only part of a schema to a Go struct.
The list of cars will have type `Car` while the parent `Person` object is `map[string]any`.
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

## How to define custom serialization and deserialization?
Currently, serialization/deserialization operations are available on maps.
This is a current design decision; it might change in the future.

```go
type Car struct {
    Name string
}

// Make sure that Car implements schema.Marshaler and schema.Unmarshaler
var (
	_ schema.Marshaler   = (*Car)(nil)
	_ schema.Unmarshaler = (*Car)(nil)
)

func (c *Car) MarshalSchema() (schema.Map, error) { // Assuming schema.Map is the intended return type, not a pointer
    return schema.MkMap(map[string]schema.Schema{
        "name": schema.MkString(c.Name),
    }), nil
}

func (c *Car) UnmarshalSchema(x schema.Map) error { // Assuming schema.Map is the intended parameter type
    for _, field := range x.Field {
        switch field.Name { // Corrected to use field.Name
        case "name":
            // Assuming field.Value is of type schema.Schema and has a MustToString method
            // or if s was meant to be field.Value of type schema.String
            if strVal, ok := field.Value.(*schema.String); ok {
                c.Name = strVal.MustToString()
            } else {
                // Handle type mismatch error for field.Value if necessary
                return fmt.Errorf("field 'name' is not a string, got %T", field.Value)
            }
        default:
            return fmt.Errorf("unknown key %s", field.Name) // Corrected to use field.Name
        }
    }
	
    return nil
}
```

### How to convert well-defined types from external packages?
```go
type Car struct {
    Name string
    LastDriven time.Time
}

// Register a conversion from time.Time to schema.String
schema.RegisterWellDefinedTypesConversion[time.Time](
  func(x time.Time) Schema {
      return MkString(x.Format(time.RFC3339Nano))
  },
  func(x Schema) time.Time {
      if v, ok := x.(*String); ok {
          t, _ := time.Parse(time.RFC3339Nano, string(*v))
          return t
      }

      panic("invalid type")
  },
)

// Then you can translate the schema back and forth without worrying about time.Time
schema := FromGo(data)
result, err := ToGoG[ExternalPackageStruct](schema)
assert.NoError(t, err)
assert.Equal(t, data, result)
```

## Roadmap
### V0.1.0
- [x] JSON <-> Schema <-> Go (with structs mapping)
- [x] Write test with wrong type conversions
- [x] Values are split into Number(Int, Float), String, Bool, and Null
- [x] Default schema registry + mkunion make union serialization/deserialization work transparently
- [x] Support pointers *string, etc.
- [x] Support DynamoDB (FromDynamoDB, ToDynamoDB)
- [x] Support for pointer to types like *string, *int, etc.
- [x] Support for relative paths like `WhenPath([]string{"*", "ListOfCars", "Car"}, UseStruct(Car{}))`. 
      Absolute paths are without `*` at the beginning.
 
### V0.2.x
- [x] Support options for `ToGo` & `FromGo` like `WithOnlyTheseRules`, `WithExtraRules`, `WithDefaultMapDef`, etc. 
      Gives better control on how schema is converted to Go.
      It's especially important from security reasons, when you want to allow only whitelisted rules for user-generated JSON input.
- [x] Schema support interface for custom type-setters that doesn't require reflection, and mkunion can leverage them. Use `UseTypeDef` eg. `WhenPath([]string{}, UseTypeDef(&someTypeDef{})),`
- [x] Support for how union names should be expressed in schema `WithUnionFormatter(func(t reflect.Type) string)`

### V0.3.x
- [x] schema.Compare method to compare two schemas

### V0.4.x
- [x] Support for Binary type
- [x] Add missing functions for MkBinary, MkFloat, MkNone

### V0.5.x
- [x] `schema.UnwrapDynamoDB` takes DynamoDB specific nesting and removes it.
- [x] Eliminate data races in `*UnionVariants[A] -> MapDefFor` & `UseUnionFormatter` data race
- [x] Introduce `ToGoG[T any]` function that makes ToGo with type assertion and tries to convert to T
- [x] Rename `schema.As` to `schema.AsDefault` and make `schema.As` a variant that returns false if the type is not supported

### V0.6.x
- [x] Support serialization of `schema.Marshaler` to `schema.Unmarshaller`, that can dramatically improve performance in some cases.
      Limitation of current implementation is that it works only on `*Map`, and doesn't allow custom serialization/deserialization on other types.
      It's not a hard decision. It's just that I don't have a use case for other types yet.

### V0.7.x
- [x] schema.Schema is now serializable and deserializable

### V0.8.x
- [x] `schema` uses `x/shape` and native types to represent variants like Map and List and Bytes
- [x] schema.Schema is refactored to leverage simpler types thanks to the `x/shape` library
- [x] schema.ToJSON and schema.FromJSON are removed and replaced by `mkunion` defaults

### V0.11.x
- [ ] `schema` becomes `data`
- [ ] data.FromGo and data.ToGo works only on primitive values
- [ ] data.FromStruct and data.ToStruct works only on structs and reflection
- [ ] data.FromDynamoDB and data.ToDynamoDB refactored