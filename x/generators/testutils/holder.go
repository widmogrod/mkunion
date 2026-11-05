package testutils

// Holder is a plain struct that holds the Tree union in its fields. It has no
// serde tag: mkunion generates its JSON serde anyway, because encoding/json on
// its own would drop the $type of the union values.
type Holder struct {
	One   Tree
	Many  []Tree
	Named map[string]Tree
	Note  string `json:"note,omitempty"`
}

// Trees is a named list of the union. It gets serde too, so a struct that
// reaches the union through it keeps $type.
type Trees []Tree

type ViaAlias struct {
	List Trees
}

// Plain holds no union, so it gets no serde.
type Plain struct {
	Note string
}
