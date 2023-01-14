# Golang recursive schema
Library allows to write code that work with any type of schemas.
Regardless if those are JSON, XML, YAML, or golang structs.

Most benefits
- Union types can be deserialized into interface field

## How to convert between json <-> go
```go
data := `{"name": "John", "cars": [{"name":"Ford"}]}`
schema := JsonToSchema(data)
nativego := SchemaToGo(schema)

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
nativego := SchemaToGo(schema, WhenPath([]string{"cars", "[*]"}, UseStruct(Car{})))

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
### V1.0.0
- [x] Write test with wrong type conversions
- [x] Value are split into Number(Int, Float), String, Bool, and Null

### V1.1.x
- [ ] Support json tags in golang to map field names to schema
- [ ] Add cata, ana, and hylo morphisms
