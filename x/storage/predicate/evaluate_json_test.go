package predicate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate/testutil"
)

func sampleDoc(t *testing.T, value testutil.SampleStruct) any {
	t.Helper()
	raw, err := shared.JSONMarshal(value)
	require.NoError(t, err)

	var doc any
	require.NoError(t, json.Unmarshal(raw, &doc))
	return doc
}

func sampleShape(t *testing.T) shape.Shape {
	t.Helper()
	s, found := shape.LookupShapeReflectAndIndex[testutil.SampleStruct]()
	require.True(t, found)
	return s
}

func TestResolveJSONPaths(t *testing.T) {
	s := sampleShape(t)

	t.Run("struct field", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, "Age")
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "Age", paths[0].String())
	})

	t.Run("explicit variant", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, `Tree["testutil.Branch"].Name`)
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "Tree.testutil.Branch.Name", paths[0].String())
	})

	t.Run("bare variant name", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, `Tree["Branch"].Name`)
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "Tree.testutil.Branch.Name", paths[0].String())
	})

	t.Run("bare union field expands only over variants that have it", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, "Tree.Name")
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "Tree.testutil.Branch.Name", paths[0].String())
	})

	t.Run("union wildcard expands over all variants with the field", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, "Tree[*].Name")
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "Tree.testutil.Branch.Name", paths[0].String())
	})

	t.Run("discriminator", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, `Tree["$type"]`)
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "Tree.$type", paths[0].String())
	})

	t.Run("list wildcard and index", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, "Friends[*].Age")
		require.NoError(t, err)
		require.Len(t, paths, 1)

		paths, err = ResolveJSONPaths(s, "Friends[0].ID")
		require.NoError(t, err)
		require.Len(t, paths, 1)
	})

	t.Run("typo in field name is an error", func(t *testing.T) {
		_, err := ResolveJSONPaths(s, "Agee")
		assert.Error(t, err)
	})

	t.Run("field missing from every variant is an error", func(t *testing.T) {
		_, err := ResolveJSONPaths(s, "Tree.DoesNotExist")
		assert.Error(t, err)
	})
}

func TestJSONEvaluator(t *testing.T) {
	doc := sampleDoc(t, testutil.SampleStruct{
		ID:      "user-1",
		Age:     39,
		Visible: true,
		Friends: []testutil.SampleStruct{
			{ID: "friend-1", Age: 20},
			{ID: "friend-2", Age: 65},
		},
		Tree: &testutil.Branch{
			Name: "root",
			Left: &testutil.Leaf{Value: schema.MkString("leaf-value")},
		},
	})

	eval := NewJSONEvaluator(sampleShape(t))

	evaluate := func(t *testing.T, where string, params ParamBinds) bool {
		t.Helper()
		pred, err := Where(where, params, nil)
		require.NoError(t, err)
		result, err := eval.Evaluate(pred.Predicate, doc, pred.Params)
		require.NoError(t, err)
		return result
	}

	t.Run("string equality", func(t *testing.T) {
		assert.True(t, evaluate(t, "ID = :id", ParamBinds{":id": schema.MkString("user-1")}))
		assert.False(t, evaluate(t, "ID = :id", ParamBinds{":id": schema.MkString("user-2")}))
	})

	t.Run("number comparison keeps fractions", func(t *testing.T) {
		assert.True(t, evaluate(t, "Age > :n", ParamBinds{":n": schema.MkFloat(38.5)}))
		assert.False(t, evaluate(t, "Age > :n", ParamBinds{":n": schema.MkFloat(39.5)}))
	})

	t.Run("bool", func(t *testing.T) {
		assert.True(t, evaluate(t, "Visible = :v", ParamBinds{":v": schema.MkBool(true)}))
	})

	t.Run("explicit union variant", func(t *testing.T) {
		assert.True(t, evaluate(t, `Tree["testutil.Branch"].Name = :n`, ParamBinds{":n": schema.MkString("root")}))
	})

	t.Run("bare union field", func(t *testing.T) {
		assert.True(t, evaluate(t, "Tree.Name = :n", ParamBinds{":n": schema.MkString("root")}))
		assert.False(t, evaluate(t, "Tree.Name = :n", ParamBinds{":n": schema.MkString("other")}))
	})

	t.Run("discriminator", func(t *testing.T) {
		assert.True(t, evaluate(t, `Tree["$type"] = :t`, ParamBinds{":t": schema.MkString("testutil.Branch")}))
		assert.False(t, evaluate(t, `Tree["$type"] = :t`, ParamBinds{":t": schema.MkString("testutil.Leaf")}))
	})

	t.Run("list wildcard matches any element", func(t *testing.T) {
		assert.True(t, evaluate(t, "Friends[*].Age > :n", ParamBinds{":n": schema.MkInt(60)}))
		assert.False(t, evaluate(t, "Friends[*].Age > :n", ParamBinds{":n": schema.MkInt(70)}))
	})

	t.Run("and or not", func(t *testing.T) {
		assert.True(t, evaluate(t, "ID = :id AND Age >= :n", ParamBinds{
			":id": schema.MkString("user-1"),
			":n":  schema.MkInt(39),
		}))
		assert.True(t, evaluate(t, "ID = :other OR Age = :n", ParamBinds{
			":other": schema.MkString("nope"),
			":n":     schema.MkInt(39),
		}))
	})

	t.Run("typo in location is an error not false", func(t *testing.T) {
		pred, err := Where("Agee = :n", ParamBinds{":n": schema.MkInt(1)}, nil)
		require.NoError(t, err)
		_, err = eval.Evaluate(pred.Predicate, doc, pred.Params)
		assert.Error(t, err)
	})
}
