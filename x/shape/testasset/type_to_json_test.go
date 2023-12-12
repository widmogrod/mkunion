package testasset

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestExampleToJSON_A(t *testing.T) {
	result, err := ExampleToJSON(&A{
		Name: "not-angel",
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"$type":"testasset.A", "testasset.A": {"name":"not-angel"}}`, string(result))

	example, err := ExampleFromJSON(result)
	assert.NoError(t, err)
	assert.Equal(t, &A{
		Name: "not-angel",
	}, example)
}

func TestExampleToJSON_B(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2023-12-08T20:09:06.523605+01:00")
	assert.NoError(t, err)

	result, err := ExampleToJSON(&B{
		Age: 123,
		A: &A{
			Name: "not-angel",
		},
		T: &now,
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{
  "$type":"testasset.B", 
  "testasset.B": {
	"age":123,
    "A": {
        "name":"not-angel"
    },
    "T": "2023-12-08T20:09:06.523605+01:00"
  }
}`, string(result))

	example, err := ExampleFromJSON(result)
	assert.NoError(t, err)
	assert.Equal(t, &B{
		Age: 123,
		A: &A{
			Name: "not-angel",
		},
		T: &now,
	}, example)
}

func TestOtherToJSON_A(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2023-12-08T20:09:06.523605+01:00")
	assert.NoError(t, err)

	result, err := SomeDSLToJSON(&Explain{
		Example: &B{
			Age: 123,
			A: &A{
				Name: "not-angel",
			},
			T: &now,
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{
  "$type": "testasset.Explain",
  "testasset.Explain": {
	"example": {
	  "$type":"testasset.B", 
	  "testasset.B": {
		"age":123,
		"A": {
			"name":"not-angel"
		},
		"T": "2023-12-08T20:09:06.523605+01:00"
	  }
	}
  }
}`, string(result))

	example, err := SomeDSLFromJSON(result)
	assert.NoError(t, err)
	assert.Equal(t, &Explain{
		Example: &B{
			Age: 123,
			A: &A{
				Name: "not-angel",
			},
			T: &now,
		},
	}, example)
}

func TestGraphDSL(t *testing.T) {
	graph := &Graph[int]{}
	graph.Vertices = map[string]*Vertex[int]{
		"1": {
			Value: 1,
			Edges: []*Edge[int]{
				{
					Weight: 1.0,
				},
			},
		},
		"2": {
			Value: 2,
			Edges: []*Edge[int]{
				{
					Weight: 2.0,
				},
			},
		},
	}

	result, err := GraphDSLToJSON[int](graph)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
  "$type": "testasset.Graph",
  "testasset.Graph": {
    "Vertices": {
      "1": {
        "Edges": [
          {
            "Weight": 1
          }
        ],
        "Value": 1
      },
      "2": {
        "Edges": [
          {
            "Weight": 2
          }
        ],
        "Value": 2
      }
    }
  }
}`, string(result))

	t.Log(string(result))

	example, err := GraphDSLFromJSON[int](result)
	assert.NoError(t, err)
	assert.Equal(t, graph, example)

}
