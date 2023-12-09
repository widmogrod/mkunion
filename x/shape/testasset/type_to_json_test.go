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
	assert.JSONEq(t, `{"$type":"github.com/widmogrod/mkunion/x/shape/testasset.A", "github.com/widmogrod/mkunion/x/shape/testasset.A": {"name":"not-angel"}}`, string(result))

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
  "$type":"github.com/widmogrod/mkunion/x/shape/testasset.B", 
  "github.com/widmogrod/mkunion/x/shape/testasset.B": {
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
  "$type": "github.com/widmogrod/mkunion/x/shape/testasset.Explain",
  "github.com/widmogrod/mkunion/x/shape/testasset.Explain": {
	"example": {
	  "$type":"github.com/widmogrod/mkunion/x/shape/testasset.B", 
	  "github.com/widmogrod/mkunion/x/shape/testasset.B": {
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
