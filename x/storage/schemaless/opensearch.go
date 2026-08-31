package schemaless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"io"
	"strings"
)

func NewOpenSearchRepository[A any](client *opensearch.Client, index string) *OpenSearchRepository[A] {
	return &OpenSearchRepository[A]{
		client:    client,
		indexName: index,
	}
}

var _ Repository[any] = (*OpenSearchRepository[any])(nil)

// OpenSearchRepository is intended as a projection / read-model store:
// data is projected into it from an authoritative store (e.g. via streams),
// so writes are expected to be re-playable.
//
// Atomicity contract: OpenSearch has no multi-document transactions, so
// UpdateRecords is atomic PER RECORD, not per batch. When a record in the
// middle of a batch fails, records processed before it stay written; the
// returned UpdateRecordsResult lists them alongside the error. Transactional
// backends (DynamoDB, in-memory) are all-or-nothing and return a nil result
// on error.
//
// Optimistic locking (PolicyIfServerNotChanged) is enforced server-side in a
// single request: a Painless scripted upsert compares the stored Version with
// the incoming one atomically on the shard and rejects stale writes with
// ErrVersionConflict. A document without a Version field is treated as absent,
// mirroring DynamoDB's "Version = :v OR attribute_not_exists(Version)".
type OpenSearchRepository[A any] struct {
	client    *opensearch.Client
	indexName string
}

// openSearchVersionConflictMarker is embedded in the script exception message
// so a version conflict can be told apart from other script failures, which
// OpenSearch reports with the same HTTP 400 status.
const openSearchVersionConflictMarker = "mkunion_version_conflict"

// openSearchVersionCheckScript atomically enforces optimistic locking and
// replaces the document content. It runs with params:
//   - doc: the full new document (already carrying the incremented Version)
//   - expected: the Version the caller last read
const openSearchVersionCheckScript = `
if (ctx._source.containsKey('Version') && ctx._source.Version != params.expected) {
  throw new IllegalArgumentException('` + openSearchVersionConflictMarker + `: stored=' + ctx._source.Version + ' expected=' + params.expected);
}
ctx._source.clear();
ctx._source.putAll(params.doc);
`

type openSearchDocMeta[A any] struct {
	Item A `json:"_source"`
}

// getDocMeta returns nil (and no error) when the document does not exist.
func (os *OpenSearchRepository[A]) getDocMeta(documentID string) (*openSearchDocMeta[Record[A]], error) {
	response, err := os.client.Get(os.indexName, documentID)
	if err != nil {
		return nil, fmt.Errorf("OpenSearchRepository.getDocMeta: request error=%s. %w", err, ErrInternalError)
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return nil, nil
	}
	if response.IsError() {
		return nil, fmt.Errorf("OpenSearchRepository.getDocMeta: response %s. %w", response.String(), ErrInternalError)
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("OpenSearchRepository.getDocMeta: read body error=%s. %w", err, ErrInternalError)
	}

	typed, err := shared.JSONUnmarshal[openSearchDocMeta[Record[A]]](result)
	if err != nil {
		return nil, fmt.Errorf("OpenSearchRepository.getDocMeta: type conversion error=%s. %w", err, ErrInvalidType)
	}

	return &typed, nil
}

func (os *OpenSearchRepository[A]) Get(recordID string, recordType RecordType) (Record[A], error) {
	meta, err := os.getDocMeta(os.recordID(recordType, recordID))
	if err != nil {
		return Record[A]{}, fmt.Errorf("OpenSearchRepository.Get: %w", err)
	}
	if meta == nil {
		return Record[A]{}, fmt.Errorf("OpenSearchRepository.Get: id=%s type=%s. %w", recordID, recordType, ErrNotFound)
	}

	return meta.Item, nil
}

func (os *OpenSearchRepository[A]) UpdateRecords(command UpdateRecords[Record[A]]) (*UpdateRecordsResult[Record[A]], error) {
	if command.IsEmpty() {
		return nil, fmt.Errorf("OpenSearchRepository.UpdateRecords: empty command. %w", ErrEmptyCommand)
	}

	// result is filled incrementally: on a mid-batch failure it is returned
	// alongside the error, listing the records that were durably written.
	result := &UpdateRecordsResult[Record[A]]{
		Saved:   make(map[string]Record[A]),
		Deleted: make(map[string]Record[A]),
	}

	for _, record := range command.Saving {
		if err := os.saveRecord(command.UpdatingPolicy, record); err != nil {
			return result, err
		}
		record.Version++
		result.Saved[record.ID] = record
	}

	for _, record := range command.Deleting {
		if err := os.deleteRecord(record); err != nil {
			return result, err
		}
		result.Deleted[record.ID] = record
	}

	return result, nil
}

func (os *OpenSearchRepository[A]) saveRecord(policy UpdatingPolicy, record Record[A]) error {
	documentID := os.toDocumentID(record)
	expectedVersion := record.Version
	record.Version++

	data, err := shared.JSONMarshal[Record[A]](record)
	if err != nil {
		return fmt.Errorf("OpenSearchRepository.saveRecord: marshal error=%s. %w", err, ErrInternalError)
	}

	if policy == PolicyOverwriteServerChanges {
		response, err := os.client.Index(os.indexName, bytes.NewReader(data), func(request *opensearchapi.IndexRequest) {
			request.DocumentID = documentID
			// make the write visible to searches immediately,
			// matching the read-after-write behaviour of the other backends
			request.Refresh = "true"
		})
		if err != nil {
			return fmt.Errorf("OpenSearchRepository.saveRecord: index error=%s. %w", err, ErrInternalError)
		}
		defer response.Body.Close()

		if response.IsError() {
			return fmt.Errorf("OpenSearchRepository.saveRecord: index response %s. %w", response.String(), ErrInternalError)
		}
		return nil
	}

	// PolicyIfServerNotChanged: one request, version check and write
	// happen atomically on the shard inside the script
	body, err := json.Marshal(map[string]any{
		"scripted_upsert": true,
		"upsert":          map[string]any{},
		"script": map[string]any{
			"lang":   "painless",
			"source": openSearchVersionCheckScript,
			"params": map[string]any{
				"doc":      json.RawMessage(data),
				"expected": int(expectedVersion),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("OpenSearchRepository.saveRecord: script marshal error=%s. %w", err, ErrInternalError)
	}

	response, err := os.client.Update(os.indexName, documentID, bytes.NewReader(body), func(request *opensearchapi.UpdateRequest) {
		request.Refresh = "true"
		// the script re-checks Version on every attempt, so retrying the
		// engine-level write race is safe
		retries := 3
		request.RetryOnConflict = &retries
	})
	if err != nil {
		return fmt.Errorf("OpenSearchRepository.saveRecord: update error=%s. %w", err, ErrInternalError)
	}
	defer response.Body.Close()

	if response.IsError() {
		responseBody, _ := io.ReadAll(response.Body)
		// a failed script version check surfaces as HTTP 400 with our marker;
		// 409 is the engine's own concurrent-write conflict
		if response.StatusCode == 409 || strings.Contains(string(responseBody), openSearchVersionConflictMarker) {
			return fmt.Errorf("OpenSearchRepository.saveRecord: id=%s. %w", record.ID, ErrVersionConflict)
		}
		return fmt.Errorf("OpenSearchRepository.saveRecord: update status=%d body=%s. %w", response.StatusCode, string(responseBody), ErrInternalError)
	}

	return nil
}

func (os *OpenSearchRepository[A]) deleteRecord(record Record[A]) error {
	response, err := os.client.Delete(os.indexName, os.toDocumentID(record), func(request *opensearchapi.DeleteRequest) {
		request.Refresh = "true"
	})
	if err != nil {
		return fmt.Errorf("OpenSearchRepository.deleteRecord: delete error=%s. %w", err, ErrInternalError)
	}
	defer response.Body.Close()

	// 404 means the record is already gone, which is the desired outcome
	if response.IsError() && response.StatusCode != 404 {
		return fmt.Errorf("OpenSearchRepository.deleteRecord: delete response %s. %w", response.String(), ErrInternalError)
	}
	return nil
}

type (
	//go:tag serde:"json"
	OpenSearchSearchResult[A any] struct {
		Hits OpenSearchSearchResultHits[A] `json:"hits"`
	}
	//go:tag serde:"json"
	OpenSearchSearchResultHits[A any] struct {
		Hits []OpenSearchSearchResultHit[A] `json:"hits"`
	}
	//go:tag serde:"json"
	OpenSearchSearchResultHit[A any] struct {
		Item A        `json:"_source"`
		Sort []string `json:"sort"`
	}
)

func (os *OpenSearchRepository[A]) FindingRecords(query FindingRecords[Record[A]]) (PageResult[Record[A]], error) {
	filters, sorters := os.toFiltersAndSorters(query)

	queryTemplate := map[string]any{}
	if query.Limit > 0 {
		if query.Limit > 0 {
			// add as last sorter _id, so that we can use search_after
			sorters = append(sorters, map[string]any{
				"_id": map[string]any{
					"order": "asc",
				},
			})
		}
		queryTemplate["size"] = query.Limit
	}

	if query.After != nil {
		afterSearch, err := shared.JSONUnmarshal[any]([]byte(*query.After))
		if err != nil {
			return PageResult[Record[A]]{}, fmt.Errorf("OpenSearchRepository.FindingRecords: after cursor unmarshal error=%s. %w", err, ErrInternalError)
		}

		queryTemplate["search_after"] = afterSearch
	}

	if len(filters) > 0 {
		queryTemplate["query"] = filters
	}
	if len(sorters) > 0 {
		queryTemplate["sort"] = sorters
	}

	body, err := json.Marshal(queryTemplate)
	if err != nil {
		return PageResult[Record[A]]{}, fmt.Errorf("OpenSearchRepository.FindingRecords: query marshal error=%s. %w", err, ErrInternalError)
	}

	log.Infof("OpenSearchRepository FindingRecords %s", string(body))

	response, err := os.client.Search(func(request *opensearchapi.SearchRequest) {
		request.Index = []string{
			os.indexName,
		}
		request.Body = bytes.NewReader(body)
	})
	if err != nil {
		return PageResult[Record[A]]{}, fmt.Errorf("OpenSearchRepository.FindingRecords: request error=%s. %w", err, ErrInternalError)
	}
	defer response.Body.Close()

	if response.IsError() {
		return PageResult[Record[A]]{}, fmt.Errorf("OpenSearchRepository.FindingRecords: response %s. %w", response.String(), ErrInternalError)
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return PageResult[Record[A]]{}, fmt.Errorf("OpenSearchRepository.FindingRecords: read body error=%s. %w", err, ErrInternalError)
	}

	hits, err := shared.JSONUnmarshal[OpenSearchSearchResult[Record[A]]](result)
	if err != nil {
		return PageResult[Record[A]]{}, fmt.Errorf("OpenSearchRepository.FindingRecords: result unmarshal error=%s. %w", err, ErrInvalidType)
	}

	var lastSort []string
	var items []Record[A]
	for _, hit := range hits.Hits.Hits {
		items = append(items, hit.Item)
		lastSort = hit.Sort
	}

	if len(items) == int(query.Limit) && lastSort != nil {
		// has next page of results
		next := query

		data, err := shared.JSONMarshal[any](lastSort)
		if err != nil {
			return PageResult[Record[A]]{}, fmt.Errorf("OpenSearchRepository.FindingRecords: after cursor marshal error=%s. %w", err, ErrInternalError)
		}
		after := string(data)
		next.After = &after

		return PageResult[Record[A]]{
			Items: items,
			Next:  &next,
		}, nil
	}

	return PageResult[Record[A]]{
		Items: items,
		Next:  nil,
	}, nil
}

func (os *OpenSearchRepository[A]) toDocumentID(record Record[A]) string {
	return os.recordID(record.Type, record.ID)
}

func (os *OpenSearchRepository[A]) recordID(recordType, recordID string) string {
	return fmt.Sprintf("%s-%s", recordType, recordID)
}

func (os *OpenSearchRepository[A]) toFiltersAndSorters(query FindingRecords[Record[A]]) (filters map[string]any, sorters []any) {
	filters = map[string]any{}
	if query.Where != nil {
		filters = os.toFilters(
			predicate.Optimize(query.Where.Predicate),
			query.Where.Params,
		)
	}

	if query.RecordType != "" {
		typeFilter := map[string]any{
			"term": map[string]any{
				"Type.keyword": query.RecordType,
			},
		}
		if len(filters) == 0 {
			filters = typeFilter
		} else {
			// keep the where-filters as one sibling clause of the type filter;
			// injecting the type term into the where bool would silence any
			// `should` clauses (a bool with a must treats should as optional)
			filters = map[string]any{
				"bool": map[string]any{
					"must": []any{filters, typeFilter},
				},
			}
		}
	}

	sorters = os.ToSorters(query.Sort)

	return
}

var mapOfOperationToOpenSearchQuery = map[string]string{
	">":  "gt",
	">=": "gte",
	"<":  "lt",
	"<=": "lte",
}

func (os *OpenSearchRepository[A]) toFilters(p predicate.Predicate, params predicate.ParamBinds) map[string]any {
	return predicate.MatchPredicateR1(
		p,
		func(x *predicate.And) map[string]any {
			var must []any
			for _, pred := range x.L {
				must = append(must, os.toFilters(pred, params))
			}
			return map[string]any{
				"bool": map[string]any{
					"must": must,
				},
			}
		},
		func(x *predicate.Or) map[string]any {
			var should []any
			for _, pred := range x.L {
				should = append(should, os.toFilters(pred, params))
			}
			return map[string]any{
				"bool": map[string]any{
					"should": should,
				},
			}
		},
		func(x *predicate.Not) map[string]any {
			return map[string]any{
				"bool": map[string]any{
					"must_not": os.toFilters(x.P, params),
				},
			}
		},
		func(x *predicate.Compare) map[string]any {
			bindValue, ok := x.BindValue.(*predicate.BindValue)
			if !ok {
				panic(fmt.Errorf("store.OpenSearchRepository.toFilters: expected bind value, got %T", x.BindValue))
			}

			bindName := bindValue.BindName
			switch x.Operation {
			case "=":
				return map[string]any{
					"term": map[string]any{
						os.termFieldName(x.Location, params[bindName]): params[bindName],
					},
				}

			case "!=":
				return map[string]any{
					"bool": map[string]any{
						"must_not": map[string]any{
							"term": map[string]any{
								os.termFieldName(x.Location, params[bindName]): params[bindName],
							},
						},
					},
				}

			case ">", ">=", "<", "<=":
				return map[string]any{
					"range": map[string]any{
						os.attrName(x.Location): map[string]any{
							mapOfOperationToOpenSearchQuery[x.Operation]: params[bindName],
						},
					},
				}
			}

			panic(fmt.Errorf("store.OpenSearchRepository.toFilters: unknown operation %s", x.Operation))
		},
	)
}

// termFieldName picks the field a term (equality) query targets. Text fields
// are matched on their exact .keyword sub-field; numeric and boolean fields
// have no .keyword sub-field, so they are matched on the field itself.
func (os *OpenSearchRepository[A]) termFieldName(location string, value schema.Schema) string {
	if _, ok := value.(*schema.String); ok {
		return fmt.Sprintf("%s.keyword", os.attrName(location))
	}
	return os.attrName(location)
}

func (os *OpenSearchRepository[A]) attrName(location string) string {
	locs, err := schema.ParseLocation(location)
	if err != nil {
		panic(err)
	}

	var result []string
	for _, loc := range locs {
		val := schema.MatchLocationR1(
			loc,
			func(x *schema.LocationField) string {
				return x.Name
			},
			func(x *schema.LocationIndex) string {
				return fmt.Sprintf("[%d]", x.Index)
			},
			func(x *schema.LocationAnything) string {
				return "schema.Map"
			},
		)
		result = append(result, val)
	}

	// TODO(schema.union) find better way to represent union map
	return strings.Join(result, ".")
}

func (os *OpenSearchRepository[A]) ToSorters(sort []SortField) []any {
	var sorters []any
	for _, s := range sort {
		if s.Descending {
			sorters = append(sorters, map[string]any{
				fmt.Sprintf("%s.keyword", os.attrName(s.Field)): map[string]any{
					"order": "desc",
				},
			})
		} else {
			sorters = append(sorters, map[string]any{
				fmt.Sprintf("%s.keyword", os.attrName(s.Field)): map[string]any{
					"order": "asc",
				},
			})
		}
	}

	return sorters
}
