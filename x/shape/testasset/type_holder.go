package testasset

// Holder is a plain struct that holds the Example union in its fields. It has
// no serde tag: mkunion generates its JSON serde anyway, because encoding/json
// on its own would drop the $type of the union values.
type Holder struct {
	One   Example
	Many  []Example
	Named map[string]Example
	Maybe *Example
	Note  string `json:"note,omitempty"`
}

// Examples is a named list of the union; ViaAlias reaches the union through it.
type Examples []Example

type ViaAlias struct {
	List Examples
}

// Plain holds no union, so it gets no serde.
type Plain struct {
	Note string
}

// OptedOut holds a union but opts out of generated serde, as a type with
// hand-written JSON methods would.
//
//go:tag serde:"-"
type OptedOut struct {
	One Example
}
