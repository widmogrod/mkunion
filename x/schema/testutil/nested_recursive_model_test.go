package testutil

import (
	"github.com/stretchr/testify/assert"
	. "github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestSchemaSchemaSerDe(t *testing.T) {
	subject := struct {
		ID     string
		Schema Schema
	}{
		ID: "foo",
		Schema: MkMap(
			MkField("name", MkString("Alpha")),
			MkField("age", MkInt(42)),
		),
	}

	expected := MkMap(
		MkField("ID", MkString("foo")),
		MkField("Schema", MkMap(
			MkField("schema.Map", MkMap(
				MkField("name", MkString("Alpha")),
				MkField("age", MkInt(42)),
			)),
		)),
	)
	schema := FromGo(subject)
	assert.Equal(t, expected, schema)

	got, err := ToGo(schema, WithExtraRules(
		WhenPath([]string{}, UseStruct(struct {
			ID     string
			Schema Schema
		}{})),
	))

	assert.NoError(t, err)
	assert.Equal(t, subject, got)
}

func TestSchemaSchemaSerDeRecursive(t *testing.T) {
	subject := &ExampleTwo{
		TwoData: MkMap(
			MkField("name", MkString("Alpha")),
			MkField("age", MkInt(42)),
		),
		TwoNext: &ExampleTwo{
			TwoData: MkInt(666),
			TwoNext: &ExampleOne{
				OneValue: "foo",
			},
		},
	}

	expected := MkMap(
		MkField("testutil.ExampleTwo", MkMap(
			MkField("TwoData", MkMap(
				MkField("schema.Map", MkMap(
					MkField("name", MkString("Alpha")),
					MkField("age", MkInt(42)),
				)),
			)),
			MkField("TwoNext", MkMap(
				MkField("testutil.ExampleTwo", MkMap(
					MkField("TwoData", MkMap(
						MkField("schema.Number", MkInt(666)),
					)),
					MkField("TwoNext", MkMap(
						MkField("testutil.ExampleOne", MkMap(
							MkField("OneValue", MkString("foo")),
						)),
					)),
				)),
			)),
		)),
	)

	schema := FromGo(subject)
	assert.Equal(t, expected, schema)

	ajson, err := ToJSON(schema)
	assert.NoError(t, err)

	bjson, err := ToJSON(expected)
	assert.NoError(t, err)

	assert.JSONEq(t, string(bjson), string(ajson))

	adata, err := FromJSON(ajson)
	assert.NoError(t, err)

	bdata, err := FromJSON(bjson)
	assert.NoError(t, err)

	assert.True(t, 0 == Compare(adata, bdata))

	got, err := ToGo(schema)
	assert.NoError(t, err)
	assert.Equal(t, subject, got)
}
func TestSchemaSchemaSerDeRecursiveList(t *testing.T) {
	subject := &ExampleTree{
		Items: []Example{
			&ExampleOne{OneValue: "foo"},
			&ExampleTwo{TwoData: MkInt(666)}},
		Schemas: []Schema{
			MkMap(
				MkField("name", MkString("Alpha")),
			),
			MkString("foo"),
		},
	}

	expected := MkMap(
		MkField("testutil.ExampleTree", MkMap(
			MkField("Items", MkList(
				MkMap(
					MkField("testutil.ExampleOne", MkMap(
						MkField("OneValue", MkString("foo")),
					)),
				),
				MkMap(
					MkField("testutil.ExampleTwo", MkMap(
						MkField("TwoData", MkMap(
							MkField("schema.Number", MkInt(666)),
						)),
						MkField("TwoNext", MkNone()),
					)),
				),
			)),
			MkField("Schemas", MkList(
				MkMap(
					MkField("schema.Map", MkMap(
						MkField("name", MkString("Alpha")),
					)),
				),
				MkMap(
					MkField("schema.String", MkString("foo")),
				),
			)),
		)),
	)

	schema := FromGo(subject)
	assert.Equal(t, expected, schema)

	ajson, err := ToJSON(schema)
	assert.NoError(t, err)

	bjson, err := ToJSON(expected)
	assert.NoError(t, err)

	assert.JSONEq(t, string(bjson), string(ajson))

	got, err := ToGo(schema)
	assert.NoError(t, err)
	assert.Equal(t, subject, got)
}
