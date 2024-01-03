package testutils

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestTree_JSON(t *testing.T) {
	var subject Tree = &Branch{
		Lit: &Leaf{Value: 111},
		List: []Tree{
			shape.Ptr(K("alpha")),
			&Ma{
				"op": &Leaf{
					Value: 222,
				},
				"to": shape.Ptr(K("beta")),
			},
			&La{
				&Leaf{Value: 333},
				shape.Ptr(K("gamma")),
			},
			&Ka{
				{
					"lp": shape.Ptr(K("delta")),
				},
				{
					"ko": shape.Ptr(K("epsilon")),
				},
			},
		},
		Map: map[string]Tree{
			"zp": &Leaf{
				Value: 444,
			},
		},
		Of: nil, // problem?
	}

	result, err := TreeToJSON(subject)
	assert.NoError(t, err)
	t.Log(string(result))

	expected := `{
  "$type": "testutils.Branch",
  "testutils.Branch": {
    "Kattr": [
      null,
      null
    ],
    "List": [
      {
        "$type": "testutils.K",
        "testutils.K": "alpha"
      },
      {
        "$type": "testutils.Ma",
        "testutils.Ma": {
          "op": {
            "$type": "testutils.Leaf",
            "testutils.Leaf": {
              "Value": 222
            }
          },
          "to": {
            "$type": "testutils.K",
            "testutils.K": "beta"
          }
        }
      },
      {
        "$type": "testutils.La",
        "testutils.La": [
          {
            "$type": "testutils.Leaf",
            "testutils.Leaf": {
              "Value": 333
            }
          },
          {
            "$type": "testutils.K",
            "testutils.K": "gamma"
          }
        ]
      },
      {
        "$type": "testutils.Ka",
        "testutils.Ka": [
          {
            "lp": {
              "$type": "testutils.K",
              "testutils.K": "delta"
            }
          },
          {
            "ko": {
              "$type": "testutils.K",
              "testutils.K": "epsilon"
            }
          }
        ]
      }
    ],
    "Lit": {
      "$type": "testutils.Leaf",
      "testutils.Leaf": {
        "Value": 111
      }
    },
    "Map": {
      "zp": {
        "$type": "testutils.Leaf",
        "testutils.Leaf": {
          "Value": 444
        }
      }
    }
  }
}
`
	assert.JSONEq(t, expected, string(result))

	output, err := TreeFromJSON([]byte(expected))
	assert.NoError(t, err)

	if diff := cmp.Diff(subject, output); diff != "" {
		t.Errorf("TreeFromJSON() mismatch (-want +got):\n%s", diff)
	}
}

func TestListOf2_SimpleType(t *testing.T) {
	subject := ListOf2[string, int]{
		ID:   "123",
		Data: "abc",
		List: []int{1, 2, 3},
		Map:  map[string]int{"a": 1, "b": 2, "c": 3},
		ListOf: ListOf[string]{
			Data: "list of string",
		},
		ListOfPtr: &ListOf[int]{
			Data: 41,
		},
	}

	result, err := subject.MarshalJSON()
	assert.NoError(t, err)
	t.Log(string(result))

	expected := `{
  "Data": "abc",
  "ID": "123",
  "List": [
    1,
    2,
    3
  ],
  "ListOf": {
    "Data": "list of string"
  },
  "ListOfPtr": {
    "Data": 41
  },
  "Time": "0001-01-01T00:00:00Z",
  "Value": null,
  "map_of_tree": {
    "a": 1,
    "b": 2,
    "c": 3
  }
}`
	assert.JSONEq(t, expected, string(result))

	output := ListOf2[string, int]{}
	err = output.UnmarshalJSON(result)
	assert.NoError(t, err)

	assert.Equal(t, subject, output)
}

func TestListOf2_ComplexType(t *testing.T) {
	//kk := K("kk")
	subject := ListOf2[Tree, ListOf[string]]{
		ListOf: ListOf[Tree]{
			Data: &Leaf{Value: 777},
		},
		ListOfPtr: &ListOf[ListOf[string]]{
			Data: ListOf[string]{
				Data: "next level",
			},
		},
	}

	result, err := subject.MarshalJSON()
	assert.NoError(t, err)

	t.Log(string(result))

	expected := `{
  "Data": null,
  "ID": "",
  "List": [],
  "ListOf": {
    "Data": {
      "$type": "testutils.Leaf",
      "testutils.Leaf": {
        "Value": 777
      }
    }
  },
  "ListOfPtr": {
    "Data": {
      "Data": "next level"
    }
  },
  "Time": "0001-01-01T00:00:00Z",
  "Value": null,
  "map_of_tree": {}
}
`
	assert.JSONEq(t, expected, string(result))

	output := &ListOf2[Tree, ListOf[string]]{}
	err = output.UnmarshalJSON([]byte(expected))
	assert.NoError(t, err)

	result2, err := output.MarshalJSON()
	assert.NoError(t, err)

	assert.JSONEq(t, expected, string(result2))
}
