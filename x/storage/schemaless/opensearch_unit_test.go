package schemaless

import (
	"io"
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

func TestOpenSearchRepository_UpdateRecords_StaleVersionConflict(t *testing.T) {
	// a failed script version check surfaces as HTTP 400 carrying the marker
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/_update/", "guarded save should use the scripted _update API")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"type":"illegal_argument_exception","reason":"mkunion_version_conflict: stored=5 expected=3"}}`))
	})

	_, err := repo.UpdateRecords(Save(Record[ExampleRecord]{
		ID:      "123",
		Type:    "ExampleRecord",
		Version: 3, // server has version 5
	}))
	assert.ErrorIs(t, err, ErrVersionConflict)
}

func TestOpenSearchRepository_UpdateRecords_ConcurrentWriteConflict(t *testing.T) {
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"type":"version_conflict_engine_exception"}}`))
	})

	_, err := repo.UpdateRecords(Save(Record[ExampleRecord]{
		ID:      "123",
		Type:    "ExampleRecord",
		Version: 3,
	}))
	assert.ErrorIs(t, err, ErrVersionConflict)
}

func TestOpenSearchRepository_UpdateRecords_GuardedSaveSendsScript(t *testing.T) {
	var gotBody []byte
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"created"}`))
	})

	result, err := repo.UpdateRecords(Save(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord", Version: 3}))
	require.NoError(t, err)
	assert.Equal(t, uint16(4), result.Saved["123:ExampleRecord"].Version)
	assert.Contains(t, string(gotBody), `"scripted_upsert":true`, "missing records must be creatable in the same request")
	assert.Contains(t, string(gotBody), `"expected":3`, "script must check the pre-increment version")
	assert.Contains(t, string(gotBody), `"Version":4`, "written document must carry the incremented version")
}

func TestOpenSearchRepository_UpdateRecords_OverwritePolicySkipsVersionCheck(t *testing.T) {
	var gotPath string
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"updated"}`))
	})

	command := Save(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord", Version: 3})
	command.UpdatingPolicy = PolicyOverwriteServerChanges

	result, err := repo.UpdateRecords(command)
	require.NoError(t, err)
	assert.Contains(t, gotPath, "/_doc/", "overwrite policy should be a plain index request, no script")
	assert.Equal(t, uint16(4), result.Saved["123:ExampleRecord"].Version)
}

func TestOpenSearchRepository_UpdateRecords_PartialResultOnMidBatchFailure(t *testing.T) {
	// one save succeeds, then the delete fails: the error must come back
	// together with a result naming what was durably written
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"created"}`))
	})

	command := Save(Record[ExampleRecord]{ID: "a", Type: "ExampleRecord"})
	command.Deleting = Delete(Record[ExampleRecord]{ID: "b", Type: "ExampleRecord"}).Deleting

	result, err := repo.UpdateRecords(command)
	assert.ErrorIs(t, err, ErrInternalError)
	require.NotNil(t, result, "partial result must accompany the error")
	assert.Len(t, result.Saved, 1, "the save before the failure was durably written")
	assert.Len(t, result.Deleted, 0, "the failed delete must not be reported")
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
