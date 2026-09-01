package schemaless

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

func osRepo(t *testing.T) *OpenSearchRepository[schema.Schema] {
	t.Helper()
	// no client: these tests exercise the pure query-building helpers
	return NewOpenSearchRepository[schema.Schema](nil, "index")
}

func TestOpenSearchToFilters(t *testing.T) {
	repo := osRepo(t)
	params := predicate.ParamBinds{
		":name": schema.MkString("Ala"),
		":age":  schema.MkInt(42),
	}

	eq := func(location, bind string) *predicate.Compare {
		return &predicate.Compare{
			Location:  location,
			Operation: "=",
			BindValue: &predicate.BindValue{BindName: predicate.BindName(bind)},
		}
	}

	t.Run("string equality targets the keyword sub-field", func(t *testing.T) {
		got, err := repo.toFilters(eq("Data.Name", ":name"), params)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"term": map[string]any{
				"Data.Name.keyword": schema.MkString("Ala"),
			},
		}, got)
	})

	t.Run("numeric equality targets the bare field", func(t *testing.T) {
		got, err := repo.toFilters(eq("Data.Age", ":age"), params)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"term": map[string]any{
				"Data.Age": schema.MkInt(42),
			},
		}, got)
	})

	t.Run("inequality wraps a must_not", func(t *testing.T) {
		cmp := eq("Data.Name", ":name")
		cmp.Operation = "!="
		got, err := repo.toFilters(cmp, params)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"bool": map[string]any{
				"must_not": map[string]any{
					"term": map[string]any{
						"Data.Name.keyword": schema.MkString("Ala"),
					},
				},
			},
		}, got)
	})

	t.Run("range operations map to gt gte lt lte", func(t *testing.T) {
		for op, osOp := range map[string]string{">": "gt", ">=": "gte", "<": "lt", "<=": "lte"} {
			cmp := eq("Data.Age", ":age")
			cmp.Operation = op
			got, err := repo.toFilters(cmp, params)
			require.NoError(t, err)
			assert.Equal(t, map[string]any{
				"range": map[string]any{
					"Data.Age": map[string]any{
						osOp: schema.MkInt(42),
					},
				},
			}, got, op)
		}
	})

	t.Run("and combines with must, or with should", func(t *testing.T) {
		and, err := repo.toFilters(&predicate.And{L: []predicate.Predicate{
			eq("Data.Name", ":name"), eq("Data.Age", ":age"),
		}}, params)
		require.NoError(t, err)
		require.Contains(t, and, "bool")
		assert.Len(t, and["bool"].(map[string]any)["must"], 2)

		or, err := repo.toFilters(&predicate.Or{L: []predicate.Predicate{
			eq("Data.Name", ":name"), eq("Data.Age", ":age"),
		}}, params)
		require.NoError(t, err)
		assert.Len(t, or["bool"].(map[string]any)["should"], 2)
	})

	t.Run("not wraps must_not", func(t *testing.T) {
		got, err := repo.toFilters(&predicate.Not{P: eq("Data.Age", ":age")}, params)
		require.NoError(t, err)
		require.Contains(t, got, "bool")
		assert.Contains(t, got["bool"].(map[string]any), "must_not")
	})

	t.Run("literal values need no binds", func(t *testing.T) {
		got, err := repo.toFilters(&predicate.Compare{
			Location:  "Data.Age",
			Operation: "=",
			BindValue: &predicate.Literal{Value: schema.MkInt(7)},
		}, nil)
		require.NoError(t, err)
		assert.Contains(t, got, "term")
	})

	t.Run("missing bind errors", func(t *testing.T) {
		_, err := repo.toFilters(eq("Data.Name", ":missing"), params)
		assert.ErrorContains(t, err, "missing bind param")
	})

	t.Run("field-to-field comparison is unsupported", func(t *testing.T) {
		_, err := repo.toFilters(&predicate.Compare{
			Location:  "Data.Name",
			Operation: "=",
			BindValue: &predicate.Locatable{Location: "Data.Other"},
		}, params)
		assert.ErrorContains(t, err, "field-to-field")
	})

	t.Run("unknown operation errors", func(t *testing.T) {
		cmp := eq("Data.Age", ":age")
		cmp.Operation = "~~"
		_, err := repo.toFilters(cmp, params)
		assert.ErrorContains(t, err, "unknown operation")
	})

	t.Run("error inside a nested branch propagates", func(t *testing.T) {
		for _, p := range []predicate.Predicate{
			&predicate.And{L: []predicate.Predicate{eq("X", ":missing")}},
			&predicate.Or{L: []predicate.Predicate{eq("X", ":missing")}},
			&predicate.Not{P: eq("X", ":missing")},
		} {
			_, err := repo.toFilters(p, params)
			assert.Error(t, err, "%T", p)
		}
	})
}

func TestOpenSearchAttrName(t *testing.T) {
	repo := osRepo(t)

	assert.Equal(t, "Data.Name", repo.attrName("Data.Name"))
	assert.Equal(t, "Data.schema.Map.Name", repo.attrName("Data[*].Name"))
	assert.Equal(t, "Items.[0]", repo.attrName("Items[0]"))
	assert.Panics(t, func() { repo.attrName("Data[") })
}

func TestOpenSearchSorters(t *testing.T) {
	repo := osRepo(t)
	// give the repo a shape where Age is a number and Name a string
	repo.shapeDef = &shape.StructLike{
		Name: "Rec",
		Fields: []*shape.FieldLike{
			{Name: "Name", Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}}},
			{Name: "Age", Type: &shape.PrimitiveLike{Kind: &shape.NumberLike{}}},
			{Name: "Ok", Type: &shape.PrimitiveLike{Kind: &shape.BooleanLike{}}},
		},
	}

	sorters := repo.ToSorters([]SortField{
		{Field: "Name", Descending: false},
		{Field: "Age", Descending: true},
		{Field: "Ok", Descending: false},
	})

	assert.Equal(t, []any{
		map[string]any{"Name.keyword": map[string]any{"order": "asc"}},
		map[string]any{"Age": map[string]any{"order": "desc"}},
		map[string]any{"Ok": map[string]any{"order": "asc"}},
	}, sorters)

	t.Run("unknown fields fall back to keyword sorting", func(t *testing.T) {
		sorters := repo.ToSorters([]SortField{{Field: "Mystery"}})
		assert.Equal(t, []any{
			map[string]any{"Mystery.keyword": map[string]any{"order": "asc"}},
		}, sorters)
	})
}

func TestShapeAtLocation(t *testing.T) {
	str := &shape.PrimitiveLike{Kind: &shape.StringLike{}}
	num := &shape.PrimitiveLike{Kind: &shape.NumberLike{}}
	nested := &shape.StructLike{
		Name:   "Inner",
		Fields: []*shape.FieldLike{{Name: "Deep", Type: num}},
	}
	root := &shape.StructLike{
		Name: "Outer",
		Fields: []*shape.FieldLike{
			{Name: "Name", Type: str},
			{Name: "Inner", Type: nested},
		},
	}

	t.Run("resolves nested struct fields", func(t *testing.T) {
		s, ok := shapeAtLocation(root, "Inner.Deep")
		require.True(t, ok)
		assert.Equal(t, num, s)
	})

	useCases := map[string]struct {
		shape    shape.Shape
		location string
	}{
		"nil shape":            {nil, "Name"},
		"unparsable location":  {root, "Name["},
		"index location":       {root, "Name[0]"},
		"wildcard location":    {root, "Name[*]"},
		"field on a primitive": {str, "Name"},
		"unknown field":        {root, "Mystery"},
	}
	for name, uc := range useCases {
		t.Run(name+" reports false", func(t *testing.T) {
			_, ok := shapeAtLocation(uc.shape, uc.location)
			assert.False(t, ok)
		})
	}
}

func TestOpenSearchToFiltersAndSorters(t *testing.T) {
	repo := osRepo(t)

	t.Run("record type alone becomes a term filter", func(t *testing.T) {
		filters, _, err := repo.toFiltersAndSorters(FindingRecords[Record[schema.Schema]]{
			RecordType: "user",
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"term": map[string]any{"Type.keyword": "user"},
		}, filters)
	})

	t.Run("where and record type combine as sibling must clauses", func(t *testing.T) {
		where := predicate.MustWhere("Data.Age > :age", predicate.ParamBinds{
			":age": schema.MkInt(1),
		}, nil)
		filters, _, err := repo.toFiltersAndSorters(FindingRecords[Record[schema.Schema]]{
			RecordType: "user",
			Where:      where,
		})
		require.NoError(t, err)
		require.Contains(t, filters, "bool")
		assert.Len(t, filters["bool"].(map[string]any)["must"], 2)
	})

	t.Run("invalid where propagates the error", func(t *testing.T) {
		where := predicate.MustWhere("Data.Age > :age", predicate.ParamBinds{
			":age": schema.MkInt(1),
		}, nil)
		where.Params = predicate.ParamBinds{} // simulate missing binds
		_, _, err := repo.toFiltersAndSorters(FindingRecords[Record[schema.Schema]]{
			Where: where,
		})
		assert.Error(t, err)
	})
}

// rtFunc lets a plain function serve as the OpenSearch HTTP transport.
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func osRepoWithResponses(t *testing.T, handler rtFunc) *OpenSearchRepository[schema.Schema] {
	t.Helper()
	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{"http://fake.invalid"},
		Transport: handler,
	})
	require.NoError(t, err)
	return NewOpenSearchRepository[schema.Schema](client, "index")
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
}

func searchHitsBody(t *testing.T, sortValues []any, records ...Record[schema.Schema]) string {
	t.Helper()
	var hits []string
	for i, rec := range records {
		source, err := shared.JSONMarshal[Record[schema.Schema]](rec)
		require.NoError(t, err)
		sort := ""
		if sortValues != nil {
			sortJSON, err := json.Marshal([]any{sortValues[i]})
			require.NoError(t, err)
			sort = fmt.Sprintf(`,"sort":%s`, sortJSON)
		}
		hits = append(hits, fmt.Sprintf(`{"_source":%s%s}`, source, sort))
	}
	return fmt.Sprintf(`{"hits":{"hits":[%s]}}`, bytes.NewBufferString(joinStrings(hits, ",")).String())
}

func joinStrings(xs []string, sep string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += sep
		}
		out += x
	}
	return out
}

func TestOpenSearchFindingRecords(t *testing.T) {
	rec := func(id string) Record[schema.Schema] {
		return Record[schema.Schema]{ID: id, Type: "user", Data: schema.MkString("d" + id), Version: 1}
	}

	t.Run("hits decode and a full page yields a next cursor that resumes", func(t *testing.T) {
		var requests []map[string]any
		repo := osRepoWithResponses(t, func(r *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(r.Body)
			var q map[string]any
			_ = json.Unmarshal(body, &q)
			requests = append(requests, q)
			return jsonResponse(200, searchHitsBody(t, []any{"s1", "s2"}, rec("1"), rec("2"))), nil
		})

		page, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{
			RecordType: "user",
			Limit:      2,
		})
		require.NoError(t, err)
		require.Len(t, page.Items, 2)
		assert.Equal(t, "1", page.Items[0].ID)
		require.NotNil(t, page.Next, "a full page must produce a next cursor")
		require.NotNil(t, page.Next.After)

		// the follow-up query must resume with search_after
		_, err = repo.FindingRecords(*page.Next)
		require.NoError(t, err)
		require.Len(t, requests, 2)
		assert.Contains(t, requests[1], "search_after")
		assert.EqualValues(t, 2, requests[0]["size"])
	})

	t.Run("short page has no next cursor", func(t *testing.T) {
		repo := osRepoWithResponses(t, func(r *http.Request) (*http.Response, error) {
			return jsonResponse(200, searchHitsBody(t, []any{"s1"}, rec("1"))), nil
		})

		page, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{Limit: 5})
		require.NoError(t, err)
		assert.Nil(t, page.Next)
	})

	t.Run("error status is an internal error", func(t *testing.T) {
		repo := osRepoWithResponses(t, func(r *http.Request) (*http.Response, error) {
			return jsonResponse(500, `{"error":"boom"}`), nil
		})

		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{})
		assert.ErrorIs(t, err, ErrInternalError)
	})

	t.Run("transport failure is an internal error", func(t *testing.T) {
		repo := osRepoWithResponses(t, func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		})

		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{})
		assert.ErrorIs(t, err, ErrInternalError)
	})

	t.Run("malformed hits payload is an invalid type error", func(t *testing.T) {
		repo := osRepoWithResponses(t, func(r *http.Request) (*http.Response, error) {
			return jsonResponse(200, `{"hits`), nil
		})

		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{})
		assert.ErrorIs(t, err, ErrInvalidType)
	})

	t.Run("malformed after cursor errors before any request", func(t *testing.T) {
		repo := osRepoWithResponses(t, func(r *http.Request) (*http.Response, error) {
			t.Fatal("no request should be sent")
			return nil, nil
		})

		after := "{broken"
		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{After: &after})
		assert.ErrorIs(t, err, ErrInternalError)
	})
}
