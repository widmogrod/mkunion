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

//go:generate go run ../../../cmd/mkunion/main.go

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

func (os *OpenSearchRepository[A]) Get(recordID string, recordType RecordType) (Record[A], error) {
	response, err := os.client.Get(os.indexName, os.recordID(recordType, recordID))
	if err != nil {
		return Record[A]{}, err
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return Record[A]{}, err
	}

	log.Println("OpenSearchRepository.GetSchema result=", string(result))

	typed, err := shared.JSONUnmarshal[OpenSearchSearchResultHit[Record[A]]](result)
	if err != nil {
		return Record[A]{}, fmt.Errorf("DynamoDBRepository.GetSchema type conversion error=%s. %w", err, ErrInvalidType)
	}

	return typed.Item, nil
}

func (os *OpenSearchRepository[A]) UpdateRecords(command UpdateRecords[Record[A]]) (*UpdateRecordsResult[Record[A]], error) {
	for _, record := range command.Saving {
		data, err := shared.JSONMarshal[Record[A]](record)
		if err != nil {
			panic(err)
		}
		_, err = os.client.Index(os.indexName, bytes.NewReader(data), func(request *opensearchapi.IndexRequest) {
			request.DocumentID = os.toDocumentID(record)
		})
		if err != nil {
			panic(err)
		}
	}

	for _, record := range command.Deleting {
		_, err := os.client.Delete(os.indexName, os.toDocumentID(record))
		if err != nil {
			panic(err)
		}
	}

	//TODO: SavingPolicy check

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

	return nil, nil
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
			panic(err)
		}

		queryTemplate["search_after"] = afterSearch
	}

	if len(filters) > 0 {
		queryTemplate["query"] = filters
	}
	if len(sorters) > 0 {
		queryTemplate["sort"] = sorters
	}

	response, err := os.client.Search(func(request *opensearchapi.SearchRequest) {
		request.Index = []string{
			os.indexName,
		}
		body, err := json.Marshal(queryTemplate)
		if err != nil {
			panic(err)
		}

		log.Infof("OpenSearchRepository FindingRecords %s", string(body))

		request.Body = bytes.NewReader(body)
	})
	if err != nil {
		panic(err)
		//return PageResult[Record[A]]{}, err
	}
	result, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
		//return PageResult[Record[A]]{}, err
	}

	hits, err := shared.JSONUnmarshal[OpenSearchSearchResult[Record[A]]](result)
	if err != nil {
		panic(err)
		//return PageResult[Record[A]]{}, err
	}
	//
	//schemed, err := schema.FromJSON(result)
	//if err != nil {
	//	panic(err)
	//	//return PageResult[Record[A]]{}, err
	//}
	//
	//hists := schema.GetSchema(schemed, "hits.hits")
	//var lastSort schema.Schema

	var lastSort []string
	var items []Record[A]
	for _, hit := range hits.Hits.Hits {
		items = append(items, hit.Item)
		lastSort = hit.Sort
	}

	//items := schema.Reduce(
	//	hits.Hits,
	//	[]Record[A]{},
	//	func(s schema.Schema, result []Record[A]) []Record[A] {
	//		typed, err := schema.ToGoG[*Record[A]](schema.GetSchema(s, "_source"))
	//		if err != nil {
	//			panic(err)
	//		}
	//		result = append(result, *typed)
	//
	//		//lastSort = schema.GetSchema(s, "sort")
	//
	//		return result
	//	})

	if len(items) == int(query.Limit) && lastSort != nil {
		// has next page of results
		next := query

		data, err := shared.JSONMarshal[any](lastSort)
		if err != nil {
			panic(err)
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
	filters = os.toFilters(
		predicate.Optimize(query.Where.Predicate),
		query.Where.Params,
	)

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
