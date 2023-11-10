package predicate

import (
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate/testutil"
	"testing"
)

func TestEvaluate(t *testing.T) {
	defValue := testutil.SampleStruct{
		ID:      "123",
		Age:     20,
		Visible: true,
		Friends: []testutil.SampleStruct{
			{
				ID:      "53",
				Age:     40,
				Visible: false,
			},
			{
				ID:      "54",
				Age:     15,
				Visible: true,
			},
		},
		Tree: &testutil.Branch{
			Name: "root",
			Left: &testutil.Branch{
				Name: "cool-branch",
				Left: &testutil.Leaf{
					Value: schema.MkInt(123),
				},
				Right: &testutil.Leaf{
					Value: schema.MkBool(true),
				},
			},
			Right: &testutil.Leaf{
				Value: schema.MkInt(123),
			},
		},
	}

	defBind := map[string]any{
		":id":             "123",
		":age":            20,
		":firstFriendId":  "53",
		":secondFriendId": "54",
		":leaf0val":       123,
		":branch1name":    "cool-branch",
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
			value:  "ID = :id AND Age <= :age AND Friends[0].ID = :firstFriendId",
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  `Tree["testutil.Branch"].Right["testutil.Leaf"].Value["schema.Number"] = :leaf0val`,
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  "Tree[*].Right[*].Value[*] = :leaf0val",
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  "Tree[*].Left[*].Name = :branch1name",
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  "Tree[*].Left[*].Left[*].Value[*] = :leaf0val",
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  `ID = "123"`,
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  `Age = 20`,
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  `Visible = true`,
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  `Friends[0].Visible = false`,
			data:   defValue,
			bind:   defBind,
			result: true,
		},
		{
			value:  "Tree[*].Left[*].Left[*].Value[*] = Tree[*].Right[*].Value[*]",
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

			schemaBind := map[BindName]schema.Schema{}
			for k, v := range uc.bind {
				schemaBind[k] = schema.FromGo(v)
			}

			if result := Evaluate(p, schema.FromGo(uc.data), schemaBind); result != uc.result {
				t.Fatalf("expected %v value, got %v value", uc.result, result)
			}
		})
	}
}
