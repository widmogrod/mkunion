---
title: Marshaling union as JSON
---

# Marshaling union as JSON

MkUnion provides you with utility functions that allow you to marshal and unmarshal union types to JSON, 
reducing the burden of writing custom marshaling and unmarshaling functions for union types.

- `shared.JSONMarshal[A any](in A) ([]byte, error)`
- `shared.JSONUnmarshal[A any](data []byte) (A, error)`

Below is an example of how to use these functions and how the output JSON looks like.


```go title="example/tree_json_test.go"
import (
    "github.com/widmogrod/mkunion/x/shared"
)

--8<-- "example/tree_json_test.go:8:30"
```

Formatted JSON output of the example above:
```json
{
  "$type": "example.Branch",
  "example.Branch": {
    "L": {
      "$type": "example.Leaf",
      "example.Leaf": {
        "Value": 1
      }
    },
    "R": {
      "$type": "example.Branch",
      "example.Branch": {
        "L": {
          "$type": "example.Branch",
          "example.Branch": {
            "L": {
              "$type": "example.Leaf",
              "example.Leaf": {
                "Value": 2
              }
            },
            "R": {
              "$type": "example.Leaf",
              "example.Leaf": {
                "Value": 3
              }
            }
          }
        },
        "R": {
          "$type": "example.Leaf",
          "example.Leaf": {
            "Value": 4
          }
        }
      }
    }
  }
}
```


There are a few things that you can notice in this example:

- Each union type has a discriminator field, `$type`, which holds the type name, and a corresponding key with the name of the type, which holds the value of the union variant.
    - This is an opinionated approach, and the library doesn't allow it to be changed.
      I was experimenting with making this behavior customizable, but it makes the code and API more complex, and I prefer to keep it simple, thereby increasing interoperability between different libraries and applications.

- Recursive union types are supported and are marshaled as nested JSON objects.

- `$type` doesn't have to have the full package import name, nor type parameter,
  mostly because in `shared.JSONUnmarshal[Tree[int]](json)` you hint that your code accepts `Tree[int]`.
    - I'm considering adding explicit type discriminators like `example.Branch[int]` or `example.Leaf[int]`.
      It could increase type strictness on the client side, but it makes generating TypeScript types more complex, and I'm not sure if it's worth it.

- It's not shown in this example, but you can also reference types and union types from other packages, and serialization will work as expected.


## Rules the generated code follows

The generated `XFromJSON`, `XToJSON`, `MarshalJSON` and `UnmarshalJSON` keep to
a few rules. They are the same rules `encoding/json` uses, so a value can move
between the two without changing the JSON on the wire.

- **A nil value is `null`.** A nil union, a nil variant pointer held in a union
  (`var t Tree = (*Leaf)(nil)`), and `(*Leaf)(nil).MarshalJSON()` all encode as
  `null`. `null` decodes to a nil union.
- **`$type` picks the variant.** When `$type` is set, the payload key is read
  for the fields. A missing payload key, or a `null` payload, is the zero value
  of that variant: `{"$type":"example.Leaf"}` decodes to `&Leaf{}`. Without
  `$type`, the first key that names a variant is used.
- **Errors say what is wrong, and can be matched.** No `$type` and no variant
  key is `shared.ErrNoType`. A `$type` the union does not have is
  `shared.ErrUnknownType`, with the name at the end of the message. Both come
  wrapped, so use `errors.Is`.
- **A type that holds a union gets serde without a tag.** `encoding/json`
  cannot keep the `$type` of a union-typed value, so when a plain struct has
  such a field (directly, behind a pointer, in a slice, array or map, or through
  a named type), or a named list or map holds the union (`type Steps []State`),
  mkunion generates its `MarshalJSON` and `UnmarshalJSON` as if it were tagged
  `//go:tag serde:"json"`. The generated file says so in a comment above the
  type. The union may live in another file or package. To keep your own
  hand-written methods instead, tag the type `//go:tag serde:"-"`. A type that
  holds no union is left to `encoding/json`, as before.
- **The methods are on the pointer.** `json.Marshal(&user)` and any value
  reached through a pointer, slice or map use them; `json.Marshal(user)` on a
  bare value does not, and a union field inside loses its `$type`. Pass a
  pointer, or use `shared.JSONMarshal[User](user)`, which does.


## Struct fields and `json` tags

Generated serialization follows the same rules as the standard library `encoding/json`:

```go
type User struct {
    ID        string  `json:"id"`             // renamed to "id"
    Password  string  `json:"-"`              // never serialized
    Nickname  string  `json:"nick,omitempty"` // dropped when empty
    Age       int     `json:",omitempty"`     // keeps name "Age", dropped when 0
    Referrer  *User   `json:"referrer"`       // nil is serialized as null
}
```

- Fields appear in JSON in **declaration order**, exactly like `encoding/json`.
- `json:"-"` skips the field on both marshal and unmarshal.
- `omitempty` drops the field when the value is empty: `false`, `0`, `""`, a `nil` pointer, or an empty slice or map.
- A `nil` pointer field without `omitempty` is serialized as `null`.

This means you can move a struct between plain `encoding/json` and mkunion serde without changing the JSON on the wire.


## Next steps

- **[Union and generic types](./generic_union.md)** - Learn about generic unions
- **[Custom Pattern Matching](./custom_pattern_matching.md)** - Learn about custom pattern matching
- **[State Machines and unions](./state_machine.md)** - Learn about modeling state machines and how union type helps