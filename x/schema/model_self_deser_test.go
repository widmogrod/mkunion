package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSchemaSchemaSerialization(t *testing.T) {
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
