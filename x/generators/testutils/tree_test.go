package testutils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
		//ID: "uuid",
		//Data: &Branch{
		//	Lit: &Leaf{Value: 111},
		//	List: []Tree{
		//		&kk,
		//	},
		//	Map: map[string]Tree{
		//		"op": &Leaf{
		//			Value: 333,
		//		},
		//	},
		//},
		//List: []ListOf[string]{
		//	{
		//		Data: "nothing happens",
		//	},
		//},
		//Map: map[Tree]ListOf[string]{
		//	&Leaf{Value: 666}: {
		//		Data: "evil",
		//	},
		//},
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
