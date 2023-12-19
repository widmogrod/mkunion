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

func NewOpenSearchRepository(client *opensearch.Client, index string) *OpenSearchRepository {
	return &OpenSearchRepository{
		client:    client,
		indexName: index,
	}
}

var _ Repository[schema.Schema] = (*OpenSearchRepository)(nil)

type OpenSearchRepository struct {
	client    *opensearch.Client
	indexName string
}

func (os *OpenSearchRepository) Get(recordID string, recordType RecordType) (Record[schema.Schema], error) {
	response, err := os.client.Get(os.indexName, os.recordID(recordType, recordID))
	if err != nil {
		return Record[schema.Schema]{}, err
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return Record[schema.Schema]{}, err
	}

	log.Println("OpenSearchRepository.Get result=", string(result))

	typed, err := shared.JSONUnmarshal[*Record[schema.Schema]](result)
	//schemed, err := schema.FromJSON(result)
	//if err != nil {
	//	return Record[schema.Schema]{}, err
	//}
	//
	//typed, err := schema.ToGoG[*Record[schema.Schema]](schema.Get(schemed, "_source"))
	if err != nil {
		return Record[schema.Schema]{}, fmt.Errorf("DynamoDBRepository.Get type conversion error=%s. %w", err, ErrInvalidType)
	}

	return *typed, nil
}

func (os *OpenSearchRepository) UpdateRecords(command UpdateRecords[Record[schema.Schema]]) error {
	for _, record := range command.Saving {
		data, err := shared.JSONMarshal[Record[schema.Schema]](record)
		//data, err := schema.ToJSON(schema.FromPrimitiveGo(record))
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

	return nil
}

type OpenSearchSearchResult struct {
	Hits struct {
		Hits []struct {
			Item Record[schema.Schema] `json:"_source"`
		}
	}
}

func (os *OpenSearchRepository) FindingRecords(query FindingRecords[Record[schema.Schema]]) (PageResult[Record[schema.Schema]], error) {
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

	//if query.After != nil {
	//	schemed, err := schema.FromJSON([]byte(*query.After))
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	list, ok := schemed.(*schema.List)
	//	if !ok {
	//		panic(fmt.Errorf("expected list, got %T", schemed))
	//	}
	//
	//	afterSearch := make([]string, len(*list))
	//	for i, item := range *list {
	//		str, ok := schema.As[string](item)
	//		if !ok {
	//			panic(fmt.Errorf("expected string, got %T", item))
	//		}
	//		afterSearch[i] = str
	//	}
	//
	//	queryTemplate["search_after"] = afterSearch
	//}

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

		log.Println("OpenSearchRepository FindingRecords ", string(body))

		request.Body = bytes.NewReader(body)
	})
	if err != nil {
		panic(err)
		//return PageResult[Record[schema.Schema]]{}, err
	}
	result, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
		//return PageResult[Record[schema.Schema]]{}, err
	}

	hits, err := shared.JSONUnmarshal[OpenSearchSearchResult](result)
	//
	//schemed, err := schema.FromJSON(result)
	//if err != nil {
	//	panic(err)
	//	//return PageResult[Record[schema.Schema]]{}, err
	//}
	//
	//hists := schema.Get(schemed, "hits.hits")
	//var lastSort schema.Schema

	var items []Record[schema.Schema]
	for _, hit := range hits.Hits.Hits {
		items = append(items, hit.Item)
	}

	//items := schema.Reduce(
	//	hits.Hits,
	//	[]Record[schema.Schema]{},
	//	func(s schema.Schema, result []Record[schema.Schema]) []Record[schema.Schema] {
	//		typed, err := schema.ToGoG[*Record[schema.Schema]](schema.Get(s, "_source"))
	//		if err != nil {
	//			panic(err)
	//		}
	//		result = append(result, *typed)
	//
	//		//lastSort = schema.Get(s, "sort")
	//
	//		return result
	//	})

	//if len(items) == int(query.Limit) && lastSort != nil {
	//	// has next page of results
	//	next := query
	//
	//	data, err := schema.ToJSON(lastSort)
	//	if err != nil {
	//		panic(err)
	//	}
	//	after := string(data)
	//	next.After = &after
	//
	//	return PageResult[Record[schema.Schema]]{
	//		Items: items,
	//		Next:  &next,
	//	}, nil
	//}

	return PageResult[Record[schema.Schema]]{
		Items: items,
		Next:  nil,
	}, nil
}

func (os *OpenSearchRepository) toDocumentID(record Record[schema.Schema]) string {
	return os.recordID(record.Type, record.ID)
}

func (os *OpenSearchRepository) recordID(recordType, recordID string) string {
	return fmt.Sprintf("%s-%s", recordType, recordID)
}

func (os *OpenSearchRepository) toFiltersAndSorters(query FindingRecords[Record[schema.Schema]]) (filters map[string]any, sorters []any) {
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

func (os *OpenSearchRepository) toFilters(p predicate.Predicate, params predicate.ParamBinds) map[string]any {
	return predicate.MustMatchPredicate(
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

func (os *OpenSearchRepository) attrName(location string) string {
	locs, err := schema.ParseLocation(location)
	if err != nil {
		panic(err)
	}

	var result []string
	for _, loc := range locs {
		val := schema.MustMatchLocation(
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

	// TODO(schema.Union) find better way to represent union map
	return strings.Join(result, ".")
}

func (os *OpenSearchRepository) ToSorters(sort []SortField) []any {
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
