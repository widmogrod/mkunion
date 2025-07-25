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



## Next steps

- **[Union and generic types](./examples/union_generic.md)** - Learn about generic unions
- **[Custom Pattern Matching](./examples/custom_pattern_matching.md)** - Learn about custom pattern matching
- **[State Machines and unions](./examples/state_machine.md)** - Learn about modeling state machines and how union type helps