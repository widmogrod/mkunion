---
title: Breaking-change check for an API
---

# Breaking-change check for an API

`mkunion breaking` compares the shapes of an API package against a snapshot
and lists what breaks clients built against that snapshot, like `buf breaking`
does for protobuf. Take the snapshot when you release, commit it next to the
API, and run the check in CI.

```bash
mkunion shape-snapshot -o shapes.json api.go     # at release: every union, variant, struct and alias
mkunion breaking --against shapes.json api.go    # later: one finding per line, exit 1 when any is breaking
```

```text
api.go:31: breaking: Request: variant ListJobs removed; clients sending it get ErrUnknownType from the server (unknown_request)
api.go:39: breaking: Search.Query: json name changed "query" → "q"; old messages carry "query" and the reader sees the zero value
api.go:65: compatible: Request: variant CancelJob added: a new operation old clients never send
```

A union whose variants embed `f.Returns[R]` is a **request** union; the union
it answers with (`answers=`, or the unions the `R`s belong to) is a **response**
union, and every type reached from one inherits its direction. The Go serde
ignores unknown keys and reads a missing key as the zero value; the exported
Swift throws `unknownType` on an unknown `$type` and makes pointers `Optional`.

| Change | Request | Response |
|---|---|---|
| variant removed | breaking | warn (never sent any more) |
| variant added | compatible | breaking (old clients throw `unknownType`) |
| variant or package renamed (`$type` changes) | breaking | breaking |
| field removed | compatible | breaking |
| field added | warn without `omitempty`, else compatible | compatible |
| json name changed | breaking | breaking |
| type changed (`string`→`int`, `T`→`[]T`) | breaking | breaking |
| integer or float width changed | warn | warn |
| `T`→`*T` | compatible | breaking (`Optional` in Swift) |
| `*T`→`T` | warn (null becomes the zero value) | compatible |
| `omitempty` added or removed | compatible | compatible |
| `f.Returns[R]` answer changed | breaking | – |
| plain struct renamed in Go | compatible | compatible |

A type reached from no union has an unknown direction and its changes are
warnings. Enum-like named types (`type Status string` plus `const`s) are
compared as their underlying type only: a removed constant is not detected.
