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

func TestOpenSearchRepository_UpdateRecords_StaleVersionConflict(t *testing.T) {
	var indexed bool
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"found":true,"_seq_no":7,"_primary_term":1,"_source":{"ID":"123","Type":"ExampleRecord","Data":{"Name":"Alice","Age":30},"Version":5}}`))
			return
		}
		indexed = true
		w.WriteHeader(http.StatusOK)
	})

	_, err := repo.UpdateRecords(Save(Record[ExampleRecord]{
		ID:      "123",
		Type:    "ExampleRecord",
		Version: 3, // server has version 5
	}))
	assert.ErrorIs(t, err, ErrVersionConflict)
	assert.False(t, indexed, "stale record should not be written")
}

func TestOpenSearchRepository_UpdateRecords_ConcurrentWriteConflict(t *testing.T) {
	var gotIfSeqNo, gotIfPrimaryTerm string
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"found":true,"_seq_no":7,"_primary_term":1,"_source":{"ID":"123","Type":"ExampleRecord","Data":{"Name":"Alice","Age":30},"Version":3}}`))
			return
		}
		gotIfSeqNo = r.URL.Query().Get("if_seq_no")
		gotIfPrimaryTerm = r.URL.Query().Get("if_primary_term")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"type":"version_conflict_engine_exception"}}`))
	})

	_, err := repo.UpdateRecords(Save(Record[ExampleRecord]{
		ID:      "123",
		Type:    "ExampleRecord",
		Version: 3,
	}))
	assert.ErrorIs(t, err, ErrVersionConflict)
	assert.Equal(t, "7", gotIfSeqNo, "write should be compare-and-swap on seq_no")
	assert.Equal(t, "1", gotIfPrimaryTerm)
}

func TestOpenSearchRepository_UpdateRecords_CreatesMissingRecord(t *testing.T) {
	var gotOpType string
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"found":false}`))
			return
		}
		gotOpType = r.URL.Query().Get("op_type")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":"created"}`))
	})

	result, err := repo.UpdateRecords(Save(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord"}))
	require.NoError(t, err)
	assert.Equal(t, "create", gotOpType, "missing record should be written as creation")
	assert.Equal(t, uint16(1), result.Saved["123"].Version)
}

func TestOpenSearchRepository_UpdateRecords_OverwritePolicySkipsVersionCheck(t *testing.T) {
	var gotRead bool
	var gotIfSeqNo string
	repo := newTestOpenSearchRepository(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gotRead = true
			return
		}
		gotIfSeqNo = r.URL.Query().Get("if_seq_no")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"updated"}`))
	})

	command := Save(Record[ExampleRecord]{ID: "123", Type: "ExampleRecord", Version: 3})
	command.UpdatingPolicy = PolicyOverwriteServerChanges

	result, err := repo.UpdateRecords(command)
	require.NoError(t, err)
	assert.False(t, gotRead, "overwrite policy should not read current version")
	assert.Empty(t, gotIfSeqNo, "overwrite policy should not use compare-and-swap")
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
