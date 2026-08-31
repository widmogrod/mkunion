package testutils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// omitFieldsMirror has the same fields and tags as OmitFields but no custom
// MarshalJSON, so encoding/json handles it natively.
type omitFieldsMirror struct {
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

func TestOmitFields_MarshalMatchesEncodingJSON_ZeroValues(t *testing.T) {
	v := OmitFields{Zeta: "z", Hidden: "secret"}
	m := omitFieldsMirror{Zeta: "z", Hidden: "secret"}

	got, err := json.Marshal(&v)
	assert.NoError(t, err)

	want, err := json.Marshal(&m)
	assert.NoError(t, err)

	assert.Equal(t, string(want), string(got))
	assert.Equal(t, `{"Zeta":"z","alpha":"","nil_ptr":null}`, string(got))
}

func TestOmitFields_MarshalMatchesEncodingJSON_FullValues(t *testing.T) {
	n := int64(42)
	v := OmitFields{
		Zeta:   "z",
		Alpha:  "a",
		Hidden: "secret",
		Note:   "n",
		Count:  7,
		Ptr:    &n,
		NilPtr: &n,
		Tags:   []string{"x", "y"},
		Extra:  map[string]string{"k": "v"},
		Flag:   true,
	}
	m := omitFieldsMirror{
		Zeta:   "z",
		Alpha:  "a",
		Hidden: "secret",
		Note:   "n",
		Count:  7,
		Ptr:    &n,
		NilPtr: &n,
		Tags:   []string{"x", "y"},
		Extra:  map[string]string{"k": "v"},
		Flag:   true,
	}

	got, err := json.Marshal(&v)
	assert.NoError(t, err)

	want, err := json.Marshal(&m)
	assert.NoError(t, err)

	assert.Equal(t, string(want), string(got))
}

func TestOmitFields_UnmarshalIgnoresDashField(t *testing.T) {
	var v OmitFields
	err := json.Unmarshal([]byte(`{"Zeta":"z","Hidden":"secret","alpha":"a"}`), &v)
	assert.NoError(t, err)
	assert.Equal(t, "z", v.Zeta)
	assert.Equal(t, "a", v.Alpha)
	assert.Equal(t, "", v.Hidden)
}

func TestOmitFields_RoundTrip(t *testing.T) {
	n := int64(9)
	v := OmitFields{Zeta: "z", Note: "n", Count: 3, Ptr: &n, Tags: []string{"t"}, Flag: true}

	data, err := json.Marshal(&v)
	assert.NoError(t, err)

	var back OmitFields
	err = json.Unmarshal(data, &back)
	assert.NoError(t, err)
	assert.Equal(t, v, back)
}
