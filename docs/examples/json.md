---
title: Marshaling union as JSON
---

# Marshaling union as JSON

MkUnion provides you with utility function that allows you to marshal and unmarshal union types to JSON, 
reducing burden of writing custom marshaling and unmarshaling functions for union types.

- `shared.JSONMarshal[A any](in A) ([]byte, error)`
- `shared.JSONUnmarshal[A any](data []byte) (A, error)`

Below is an example of how to use those functions and how the output JSON looks like.


```go title="example/tree_json_test.go"
import (
    "github.com/widmogrod/mkunion/x/shared"
)

--8<-- "example/tree_json_test.go:8:30"
```

Formated JSON output of the example above:
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


There are few things that you can notice in this example:

- Each union type discriminator field `$type` field that holds the type name, and corresponding key with the name of the type, that holds value of union variant.
    - This is opinionated way, and library don't allow to change it.
      I was experimenting with making this behaviour customizable, but it make code and API mode complex, and I prefer to keep it simple, and increase interoperability between different libraries and applications, that way.

- Recursive union types are supported, and they are marshaled as nested JSON objects.]

- `$type` don't have to have full package import name, nor type parameter,
  mostly because in `shared.JSONUnmarshal[Tree[int]](json)` you hint that your code accepts `Tree[int]`.
    - I'm considering adding explicit type discriminators like `example.Branch[int]` or `example.Leaf[int]`.
      It could increase type strictness on client side, BUT it makes generating TypeScript types more complex, and I'm not sure if it's worth it.

- It's not shown on this example, but you can also reference types and union types from other packages, and serialization will work as expected.