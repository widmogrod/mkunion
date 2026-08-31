package testutils

// OmitFields exercises encoding/json tag compatibility:
// field declaration order, json:"-", and omitempty.
//
//go:tag serde:"json"
type OmitFields struct {
	Zeta   string
	Alpha  string            `json:"alpha"`
	Hidden string            `json:"-"`
	Note   string            `json:"note,omitempty"`
	Count  int               `json:",omitempty"`
	Ptr    *int64            `json:"ptr,omitempty"`
	NilPtr *int64            `json:"nil_ptr"`
	Tags   []string          `json:"tags,omitempty"`
	Extra  map[string]string `json:"extra,omitempty"`
	Flag   bool              `json:"flag,omitempty"`
}
