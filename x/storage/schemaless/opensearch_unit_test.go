package schemaless

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOpenSearchRepository(t *testing.T, handler http.HandlerFunc) *OpenSearchRepository[ExampleRecord] {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{server.URL},
	})
	require.NoError(t, err)

	return NewOpenSearchRepository[ExampleRecord](client, "test-index")
}

func TestOpenSearchRepository_Get_NotFound(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"_index":"test-index","_id":"ExampleRecord-123","found":false}`))
	})

	_, err := repo.Get("123", "ExampleRecord")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestOpenSearchRepository_Get_ServerError(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})

	_, err := repo.Get("123", "ExampleRecord")
	assert.ErrorIs(t, err, ErrInternalError)
}

func TestOpenSearchRepository_Get_Found(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"_source":{"ID":"123","Type":"ExampleRecord","Data":{"Name":"Alice","Age":30},"Version":1}}`))
	})

	record, err := repo.Get("123", "ExampleRecord")
	require.NoError(t, err)
	assert.Equal(t, "Alice", record.Data.Name)
	assert.Equal(t, uint16(1), record.Version)
}

func TestOpenSearchRepository_UpdateRecords_EmptyCommand(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected")
	})

	_, err := repo.UpdateRecords(UpdateRecords[Record[ExampleRecord]]{})
	assert.ErrorIs(t, err, ErrEmptyCommand)
}

func TestOpenSearchRepository_UpdateRecords_VersionConflict(t *testing.T) {
	var gotVersion, gotVersionType string
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		gotVersion = r.URL.Query().Get("version")
		gotVersionType = r.URL.Query().Get("version_type")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"type":"version_conflict_engine_exception"}}`))
	})

	_, err := repo.UpdateRecords(Save(Record[ExampleRecord]{
		ID:      "123",
		Type:    "ExampleRecord",
		Version: 3,
	}))
	assert.ErrorIs(t, err, ErrVersionConflict)
	assert.Equal(t, "4", gotVersion, "should send incremented version for optimistic locking")
	assert.Equal(t, "external", gotVersionType)
}

func TestOpenSearchRepository_UpdateRecords_OverwritePolicySkipsVersionCheck(t *testing.T) {
	var gotVersionType string
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		gotVersionType = r.URL.Query().Get("version_type")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"updated"}`))
	})

	command := Save(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord", Version: 3})
	command.UpdatingPolicy = PolicyOverwriteServerChanges

	result, err := repo.UpdateRecords(command)
	require.NoError(t, err)
	assert.Empty(t, gotVersionType, "overwrite policy should not use versioning")
	assert.Equal(t, uint16(4), result.Saved["123"].Version)
}

func TestOpenSearchRepository_UpdateRecords_ServerError(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})

	_, err := repo.UpdateRecords(Save(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord"}))
	assert.ErrorIs(t, err, ErrInternalError)
}

func TestOpenSearchRepository_UpdateRecords_DeleteMissingRecordIsOK(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"result":"not_found"}`))
	})

	result, err := repo.UpdateRecords(Delete(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord"}))
	require.NoError(t, err)
	assert.Len(t, result.Deleted, 1)
}

func TestOpenSearchRepository_UpdateRecords_DeleteServerError(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})

	_, err := repo.UpdateRecords(Delete(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord"}))
	assert.ErrorIs(t, err, ErrInternalError)
}

func TestOpenSearchRepository_FindingRecords_ServerError(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})

	_, err := repo.FindingRecords(FindingRecords[Record[ExampleRecord]]{})
	assert.ErrorIs(t, err, ErrInternalError)
}

func TestOpenSearchRepository_FindingRecords_BadAfterCursor(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected")
	})

	after := "not-json"
	_, err := repo.FindingRecords(FindingRecords[Record[ExampleRecord]]{After: &after})
	assert.ErrorIs(t, err, ErrInternalError)
}
