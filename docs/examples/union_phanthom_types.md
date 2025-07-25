---
title: Phantom types use cases
---

Leverage Go's type system for compile-time guarantees.

## Measurements as phantom types

Phantom types for units COULD help prevents catastrophic bugs like the Mars Climate Orbiter ($327M loss due to metric/imperial confusion).
The type system won't let you mix incompatible units.

```go title="example/units.go"
// 
--8<-- "./example/units.go:example"
```

## State tracking and phantom types

```go title="example/connection.go"
--8<-- "./example/connection.go:example"
```

