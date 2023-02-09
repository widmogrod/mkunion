package schema

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

type someJson struct {
	S string
	I int
	F float64
	B bool
	L []*someJson
	M map[string]*someJson
}

var (
	_ TypeMapDefinition = (*someTypeDef)(nil)
	_ MapBuilder        = (*someTypeDef)(nil)
)

type someTypeDef struct {
	v *someJson
}

func (s *someTypeDef) Set(key string, value any) error {
	switch key {
	case "S":
		s.v.S = value.(string)
	case "I":
		switch x := value.(type) {
		case float64:
			s.v.I = int(x)
		case int:
			s.v.I = x
		}
	case "F":
		s.v.F = value.(float64)
	case "B":
		s.v.B = value.(bool)
	case "L":
		switch x := value.(type) {
		case []interface{}:
			s.v.L = make([]*someJson, len(x))
			for i, v := range x {
				s.v.L[i] = v.(*someJson)
			}

		case []*someJson:
			s.v.L = x
		}
	case "M":
		switch x := value.(type) {
		case map[string]interface{}:
			s.v.M = make(map[string]*someJson, len(x))
			for k, v := range x {
				s.v.M[k] = v.(*someJson)
			}
		case map[string]*someJson:
			s.v.M = x

		}
	}

	return nil
}

func (s *someTypeDef) Build() any {
	return s.v
}

func (s *someTypeDef) NewMapBuilder() MapBuilder {
	return &someTypeDef{
		v: &someJson{},
	}
}

var (
	benchmarkToGo any = nil
	unmarshalJSON     = []byte(`{
		"S": "string",
		"I": 1,	
		"F": 1.1,
		"B": true,
		"L": [
			{"S": "string", "I": 2, "F": 2.1, "B": true, "M": {"key": {"S": "string", "I": 3, "F": 3.1, "B": true}}},
			{"S": "string", "I": 4, "F": 4.1, "B": true, "L": [{"S": "string", "I": 5, "F": 5.1, "B": true}]}
		],
		"M": {
			"key": {"S": "string", "I": 6, "F": 6.1, "B": true}
		}
	}`)

	expectedSomeStruct = &someJson{
		S: "string",
		I: 1,
		F: 1.1,
		B: true,
		L: []*someJson{
			{
				S: "string",
				I: 2,
				F: 2.1,
				B: true,
				M: map[string]*someJson{
					"key": {
						S: "string",
						I: 3,
						F: 3.1,
						B: true,
					},
				},
			},
			{
				S: "string",
				I: 4,
				F: 4.1,
				B: true,
				L: []*someJson{
					{
						S: "string",
						I: 5,
						F: 5.1,
						B: true,
					},
				},
			},
		},
		M: map[string]*someJson{
			"key": {
				S: "string",
				I: 6,
				F: 6.1,
				B: true,
			},
		},
	}
)

func Benchmark_Json_Unmarshal_Native(b *testing.B) {
	var r any
	var err error
	for i := 0; i < b.N; i++ {
		err = json.Unmarshal(unmarshalJSON, &r)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}
	}
	benchmarkToGo = r
}

func Benchmark_Json_Unmarshal_Schema(b *testing.B) {
	var r any
	var err error
	for i := 0; i < b.N; i++ {
		r, err = FromJSON(unmarshalJSON)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}
	}
	benchmarkToGo = r
}

func Benchmark_Json_Unmarshal_Struct_Native(b *testing.B) {
	var r any
	var err error
	for i := 0; i < b.N; i++ {
		r = someJson{}
		err = json.Unmarshal(unmarshalJSON, &r)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}
	}
	benchmarkToGo = r
}

func Benchmark_Json_Unmarshal_Struct_Schema(b *testing.B) {
	var r any
	var schema Schema
	var err error
	rule := WithOnlyTheseRules(
		WhenPath([]string{}, UseStruct(&someJson{})),
		WhenPath([]string{"*", "L", "[*]"}, UseStruct(&someJson{})),
		WhenPath([]string{"*", "M", "*"}, UseStruct(&someJson{})),
	)
	for i := 0; i < b.N; i++ {
		schema, err = FromJSON(unmarshalJSON)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}

		r, err = ToGo(schema, rule)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}
	}
	benchmarkToGo = r
}
func Benchmark_Json_Unmarshal_Struct_TypeDef_Schema(b *testing.B) {
	var r any
	var schema Schema
	var err error
	rule := WithOnlyTheseRules(
		WhenPath([]string{}, UseTypeDef(&someTypeDef{})),
		WhenPath([]string{"*", "L", "[*]"}, UseTypeDef(&someTypeDef{})),
		WhenPath([]string{"*", "M", "*"}, UseTypeDef(&someTypeDef{})),
	)
	for i := 0; i < b.N; i++ {
		schema, err = FromJSON(unmarshalJSON)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}

		r, err = ToGo(schema, rule)
		if err != nil {
			b.Fatalf("failed: %v", err)
		}
	}
	benchmarkToGo = r
}

func TestSomeTypeUseTypeDef(t *testing.T) {
	var r any
	var schema Schema
	var err error
	rule := WithOnlyTheseRules(
		WhenPath([]string{}, UseTypeDef(&someTypeDef{})),
		WhenPath([]string{"*", "L", "[*]"}, UseTypeDef(&someTypeDef{})),
		WhenPath([]string{"*", "M", "*"}, UseTypeDef(&someTypeDef{})),
	)
	schema, err = FromJSON(unmarshalJSON)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	r, err = ToGo(schema, rule)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	assert.Equal(t, expectedSomeStruct, r)
}
func TestSomeType(t *testing.T) {
	rule := WithOnlyTheseRules(
		WhenPath([]string{}, UseStruct(&someJson{})),
		WhenPath([]string{"*", "L", "[*]"}, UseStruct(&someJson{})),
		WhenPath([]string{"*", "M", "*"}, UseStruct(&someJson{})),
	)
	schema, err := FromJSON(unmarshalJSON)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	r, err := ToGo(schema, rule)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	assert.Equal(t, expectedSomeStruct, r)
}
