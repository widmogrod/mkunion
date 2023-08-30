package predicate

import (
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

type sampleStruct struct {
	ID      string
	Age     int
	Friends []sampleStruct
}

func TestEvaluate(t *testing.T) {
	defValue := sampleStruct{
		ID:  "123",
		Age: 20,
		Friends: []sampleStruct{
			{
				ID:  "53",
				Age: 40,
			},
			{
				ID:  "54",
				Age: 15,
			},
		},
	}

	defBind := map[string]any{
		":id":             "123",
		":age":            20,
		":firstFriendId":  "53",
		":secondFriendId": "54",
	}

	useCases := []struct {
		value  string
		data   any
		bind   map[string]any
		result bool
	}{
		{
			value:  "ID = :id",
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  "ID = :id AND Age <= :age",
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  "ID = :id AND Age <= :age AND Friends.[0].ID = :firstFriendId",
			data:   defValue,
			bind:   defBind,
			result: true,
		},
	}
	for _, uc := range useCases {
		t.Run(uc.value, func(t *testing.T) {
			p, err := Parse(uc.value)
			if err != nil {
				t.Fatal(err)
			}

			schemaBind := map[string]schema.Schema{}
			for k, v := range uc.bind {
				schemaBind[k] = schema.FromGo(v)
			}

			if result := Evaluate(p, schema.FromGo(uc.data), schemaBind); result != uc.result {
				t.Fatalf("expected %v value, got %v value", uc.result, result)
			}
		})
	}

}
