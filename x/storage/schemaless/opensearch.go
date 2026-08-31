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

type OpenSearchRepository[A any] struct {
	client    *opensearch.Client
	indexName string
}

// openSearchDocMeta carries the document source together with the
// concurrency-control metadata needed for compare-and-swap writes.
type openSearchDocMeta[A any] struct {
	Item        A    `json:"_source"`
	SeqNo       *int `json:"_seq_no"`
	PrimaryTerm *int `json:"_primary_term"`
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

	for _, record := range command.Saving {
		documentID := os.toDocumentID(record)

		// Mimic the DynamoDB backend's optimistic locking condition
		// "Version = :version OR attribute_not_exists(Version)":
		// when the document exists its stored version must match the record's,
		// and the write is a compare-and-swap on the document's sequence number;
		// when it does not exist the write must be a creation.
		var current *openSearchDocMeta[Record[A]]
		if command.UpdatingPolicy == PolicyIfServerNotChanged {
			var err error
			current, err = os.getDocMeta(documentID)
			if err != nil {
				return nil, fmt.Errorf("OpenSearchRepository.UpdateRecords: %w", err)
			}
			if current != nil && current.Item.Version != record.Version {
				return nil, fmt.Errorf("OpenSearchRepository.UpdateRecords: id=%s stored version %d != %d. %w",
					record.ID, current.Item.Version, record.Version, ErrVersionConflict)
			}
		}

		record.Version++
		data, err := shared.JSONMarshal[Record[A]](record)
		if err != nil {
			return nil, fmt.Errorf("OpenSearchRepository.UpdateRecords: marshal error=%s. %w", err, ErrInternalError)
		}
		response, err := os.client.Index(os.indexName, bytes.NewReader(data), func(request *opensearchapi.IndexRequest) {
			request.DocumentID = documentID
			// make the write visible to searches immediately,
			// matching the read-after-write behaviour of the other backends
			request.Refresh = "true"
			if command.UpdatingPolicy == PolicyIfServerNotChanged {
				if current == nil {
					request.OpType = "create"
				} else {
					request.IfSeqNo = current.SeqNo
					request.IfPrimaryTerm = current.PrimaryTerm
				}
			}
		})
		if err != nil {
			return nil, fmt.Errorf("OpenSearchRepository.UpdateRecords: index error=%s. %w", err, ErrInternalError)
		}
		err = func() error {
			defer response.Body.Close()
			if response.StatusCode == 409 {
				return fmt.Errorf("OpenSearchRepository.UpdateRecords: id=%s. %w", record.ID, ErrVersionConflict)
			}
			if response.IsError() {
				return fmt.Errorf("OpenSearchRepository.UpdateRecords: index response %s. %w", response.String(), ErrInternalError)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	for _, record := range command.Deleting {
		response, err := os.client.Delete(os.indexName, os.toDocumentID(record), func(request *opensearchapi.DeleteRequest) {
			request.Refresh = "true"
		})
		if err != nil {
			return nil, fmt.Errorf("OpenSearchRepository.UpdateRecords: delete error=%s. %w", err, ErrInternalError)
		}
		err = func() error {
			defer response.Body.Close()
			// 404 means the record is already gone, which is the desired outcome
			if response.IsError() && response.StatusCode != 404 {
				return fmt.Errorf("OpenSearchRepository.UpdateRecords: delete response %s. %w", response.String(), ErrInternalError)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	result := &UpdateRecordsResult[Record[A]]{
		Saved:   make(map[string]Record[A]),
		Deleted: make(map[string]Record[A]),
	}

	for _, value := range command.Saving {
		value.Version++
		result.Saved[value.ID] = value
	}

	for _, value := range command.Deleting {
		result.Deleted[value.ID] = value
	}

	return result, nil
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
		if filters["bool"] == nil {
			filters["bool"] = map[string]any{}
		}
		if filters["bool"].(map[string]any)["must"] == nil {
			filters["bool"].(map[string]any)["must"] = []any{}
		}
		filters["bool"].(map[string]any)["must"] = append(filters["bool"].(map[string]any)["must"].([]any), map[string]any{
			"term": map[string]any{
				"Type.keyword": query.RecordType,
			},
		})
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
						fmt.Sprintf("%s.keyword", os.attrName(x.Location)): params[bindName],
					},
				}

			case "!=":
				return map[string]any{
					"bool": map[string]any{
						"must_not": map[string]any{
							"term": map[string]any{
								fmt.Sprintf("%s.keyword", os.attrName(x.Location)): params[bindName],
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
