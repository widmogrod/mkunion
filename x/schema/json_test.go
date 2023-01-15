package schema

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJsonSchema(t *testing.T) {
	useCases := map[string]struct {
		in []byte
	}{
		"simple": {
			in: []byte(`{"Foo": 1, "Bar": 2}`),
		},
		"nested": {
			in: []byte(`{"Foo": 1, "Bar": {"Foo": 1, "Bar": 2}}`),
		},
		"array": {
			in: []byte(`{"Foo": 1, "Bar": [{"Foo": 1, "Bar": 2}]}`),
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			schema, err := FromJSON(uc.in)
			assert.NoError(t, err)
			data, err := ToJSON(schema)
			assert.NoError(t, err)
			assert.JSONEq(t, string(uc.in), string(data))
		})
	}
}

type AStruct struct {
	Foo float64 `json:"foo"`
	Bar float64 `json:"bar"`
}

type BStruct struct {
	Foo  float64 `json:"foo"`
	Bars []string
}

func TestJsonToSchema(t *testing.T) {
	json := []byte(`{"Foo": 1, "Bar": 2}`)
	schema, err := FromJSON(json)
	assert.NoError(t, err)

	gonative := ToGo(schema)
	assert.Equal(t, map[string]interface{}{
		"Foo": float64(1),
		"Bar": float64(2),
	}, gonative)

	gostruct := ToGo(
		schema,
		WhenPath([]string{}, UseStruct(AStruct{})),
	)
	assert.Equal(t, AStruct{
		Foo: 1,
		Bar: 2,
	}, gostruct)
}

type SomeOneOf struct {
	A *TestStruct1
	B *TestStruct2
}

func TestOneOfJSON(t *testing.T) {
	in := &SomeOneOf{
		A: &TestStruct1{
			Bar: "bar",
		},
		B: &TestStruct2{
			Baz: "baz",
		},
	}

	data, err := json.Marshal(in)
	assert.NoError(t, err)

	t.Log(string(data))
	assert.JSONEq(t,
		`{"A":{"Foo":0,"Bar":"bar","Other":null},"B":{"Baz":"baz","Count":0}}`,
		string(data))

	sch, err := FromJSON(data)
	assert.NoError(t, err)

	backToJSON, err := ToJSON(sch)
	assert.NoError(t, err)

	assert.JSONEq(t,
		`{"A":{"Foo":0,"Bar":"bar","Other":null},"B":{"Baz":"baz","Count":0}}`,
		string(backToJSON))

	out := ToGo(sch,
		WhenPath([]string{}, UseStruct(&SomeOneOf{})),
		WhenPath([]string{"A"}, UseStruct(&TestStruct1{})),
		WhenPath([]string{"B"}, UseStruct(&TestStruct2{})),
	)
	assert.Equal(t, in, out)
}
