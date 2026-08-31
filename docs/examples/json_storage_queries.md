---
title: Querying unions stored as JSON
---
# Querying unions stored as JSON

The `x/storage/schemaless/jsonful` package stores records in the plain mkunion
JSON encoding — the same bytes `shared.JSONMarshal` produces — and lets you
query them with locations that mirror your Go types.

Compared to the `schema.Schema`-based repository:

- records are encoded **once, at write time**; queries never marshal or reflect
- query locations are **validated against the type's shape** — a typo is an
  error, not an empty result
- union fields can be queried **without naming the variant**; the query
  compiler expands the location over every variant that has the field

## Creating a repository

```go
import (
    "github.com/widmogrod/mkunion/x/storage/schemaless"
    "github.com/widmogrod/mkunion/x/storage/schemaless/jsonful"
)

repo, err := jsonful.NewInMemoryRepository[workflow.State]()
if err != nil {
    // the shape of workflow.State must be registered,
    // which mkunion-generated code does in init()
}
```

`InMemoryRepository[T]` implements `schemaless.Repository[T]`, so it is a
drop-in replacement anywhere the typed repository is used — including
`taskqueue.NewTaskQueue`.

## Saving records

```go
_, err = repo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
    ID:   "run-1",
    Type: "process",
    Data: state,
}))
```

Optimistic concurrency rules:

- a **new** record must be saved with `Version: 0`; any other version is a
  `schemaless.ErrVersionConflict`
- an **update** must carry the version it read; a stale version is a conflict,
  and a failed command leaves the store untouched
- `PolicyOverwriteServerChanges` skips the version check and overwrites

## The query language

Locations are written the way you read your Go types. Given:

```go
//go:tag mkunion:"State"
type (
    Done  struct { Result schema.Schema; BaseState BaseState }
    Error struct { Code string; Retried int64; BaseState BaseState }
    Await struct { CallbackID string; BaseState BaseState }
)
```

| Location | Meaning |
|---|---|
| `Data.BaseState.RunID` | bare field; expands over **every** variant that has it |
| `Data["workflow.Await"].CallbackID` | this variant only |
| `Data["Await"].CallbackID` | short variant name works too |
| `Data["$type"]` | the union discriminator, e.g. `"workflow.Await"` |
| `Data[*].BaseState.RunID` | explicit wildcard over all variants |
| `Friends[*].Age`, `Friends[0].ID` | list wildcard and index |
| `ID`, `Type`, `Version` | record metadata |

Example:

```go
result, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
    RecordType: "process",
    Where: predicate.MustWhere(
        `Data["workflow.Await"].CallbackID = :callbackID`,
        predicate.ParamBinds{
            ":callbackID": schema.MkString(callbackID),
        }, nil),
    Limit: 1,
})
```

### Why bare fields matter

Before, filtering "all states of flow X" meant an OR over every state variant:

```
Data["workflow.Done"]["BaseState"]["Flow"]["workflow.Flow"]["Name"] = :name
OR Data["workflow.Error"]["BaseState"]["Flow"]["workflow.Flow"]["Name"] = :name
OR ... (one branch per variant)
```

Now one location does it, and a variant added tomorrow is included
automatically:

```
Data.BaseState.Flow.Name = :name
```

This keeps the exhaustiveness promise unions make: the compiler, not you,
enumerates the variants.

### Errors instead of empty results

A location that does not exist in the type fails the query:

```go
_, err = repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
    Where: predicate.MustWhere(`Data.BaseState.RunIDD = :x`, ...),
})
// err: union workflow.State: no variant (...) has BaseState.RunIDD
```

Number comparisons are exact (`1.9 > 1.2` is true; the old `schema.Compare`
truncated to integers). Sorting is deterministic: records tie-break on `ID`,
so page cursors are stable even without an explicit sort.

## Change streams with the same language

`repo.AppendLog()` returns an append log whose `Subscribe` filter uses the same
locations, validated up front:

```go
stream := repo.AppendLog()
err := stream.Subscribe(ctx, 0,
    predicate.MustWhere(
        `Data["workflow.Error"].Retried < Data["workflow.Error"].BaseState.DefaultMaxRetries`,
        predicate.ParamBinds{}, nil),
    func(change schemaless.Change[workflow.State]) {
        // react to matching changes
    })
```

A filter with a bad location returns an error from `Subscribe` instead of
silently dropping every change.

## Complete example

`example/my-app` runs entirely on this storage: workflow execution, the
callback lookup, the scheduled/timeout selectors, and the retry stream all use
these queries against `jsonful.NewInMemoryRepository[workflow.State]()`.
See `x/storage/schemaless/jsonful/repo_test.go` for focused examples of every
query form.
