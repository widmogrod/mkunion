package schemaless

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"strings"
)

func NewDynamoDBRepository(client *dynamodb.Client, tableName string) *DynamoDBRepository {
	return &DynamoDBRepository{
		client:    client,
		tableName: tableName,
	}
}

var _ Repository[schema.Schema] = (*DynamoDBRepository)(nil)

type DynamoDBRepository struct {
	client    *dynamodb.Client
	tableName string
}

func (d *DynamoDBRepository) Get(key, recordType string) (Record[schema.Schema], error) {
	item, err := d.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{
				Value: key,
			},
			"Type": &types.AttributeValueMemberS{
				Value: recordType,
			},
		},
		TableName:      &d.tableName,
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return Record[schema.Schema]{}, fmt.Errorf("DynamoDBRepository.Get error=%s. %w", err, ErrInternalError)
	}

	if len(item.Item) == 0 {
		return Record[schema.Schema]{}, fmt.Errorf("DynamoDBRepository.Get not found. %w", ErrNotFound)
	}

	i := &types.AttributeValueMemberM{
		Value: item.Item,
	}

	schemed, err := schema.FromDynamoDB(i)
	if err != nil {
		return Record[schema.Schema]{}, fmt.Errorf("DynamoDBRepository.Get schema conversion error=%s. %w", err, ErrInternalError)
	}

	typed, err := schema.ToGoG[*Record[schema.Schema]](schemed)
	if err != nil {
		return Record[schema.Schema]{}, fmt.Errorf("DynamoDBRepository.Get type conversion error=%s. %w", err, ErrInvalidType)
	}

	return *typed, nil
}

func (d *DynamoDBRepository) UpdateRecords(command UpdateRecords[Record[schema.Schema]]) error {
	if command.IsEmpty() {
		return fmt.Errorf("DynamoDBRepository.UpdateRecords: empty command %w", ErrEmptyCommand)
	}

	var transact []types.TransactWriteItem
	for _, value := range command.Saving {
		originalVersion := value.Version
		value.Version++
		sch := schema.FromPrimitiveGo(value)
		item := schema.ToDynamoDB(sch)
		if _, ok := item.(*types.AttributeValueMemberM); !ok {
			return fmt.Errorf("DynamoDBRepository.UpdateRecords: unsupported type: %T", item)
		}

		final, ok := item.(*types.AttributeValueMemberM)
		if !ok {
			return fmt.Errorf("DynamoDBRepository.UpdateRecords: expected map as item. %w", ErrInternalError)
		}

		transact = append(transact, types.TransactWriteItem{
			Put: &types.Put{
				TableName:           aws.String(d.tableName),
				Item:                final.Value,
				ConditionExpression: aws.String("Version = :version OR attribute_not_exists(Version)"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":version": &types.AttributeValueMemberN{
						Value: fmt.Sprintf("%d", originalVersion),
					},
				},
			},
		})
	}

	for _, id := range command.Deleting {
		transact = append(transact, types.TransactWriteItem{
			Delete: &types.Delete{
				TableName: aws.String(d.tableName),
				Key: map[string]types.AttributeValue{
					"ID": &types.AttributeValueMemberS{
						Value: id.ID,
					},
					"Type": &types.AttributeValueMemberS{
						Value: id.Type,
					},
				},
			},
		})
	}

	_, err := d.client.TransactWriteItems(context.Background(), &dynamodb.TransactWriteItemsInput{
		TransactItems: transact,
	})

	if err != nil {
		respErr := &http.ResponseError{}
		if errors.As(err, &respErr) {
			conditional := &types.TransactionCanceledException{}
			if errors.As(respErr.ResponseError.Err, &conditional) {
				for _, reason := range conditional.CancellationReasons {
					if *reason.Code == "ConditionalCheckFailed" {
						return fmt.Errorf("store.DynamoDBRepository.UpdateRecords: %w", ErrVersionConflict)
					}
				}
			}
		}
		return err
	}

	return nil
}

func (d *DynamoDBRepository) FindingRecords(query FindingRecords[Record[schema.Schema]]) (PageResult[Record[schema.Schema]], error) {
	filterExpression, paramsExpression, expressionNames, err := d.buildFilterExpression(query)
	if err != nil {
		return PageResult[Record[schema.Schema]]{}, err
	}

	log.Infof("\nfilterExpression: %#v \n", filterExpression)
	for k, v := range paramsExpression {
		log.Infof("paramsExpression[%s]: %#v \n", k, v)
	}

	for k, v := range expressionNames {
		log.Infof("expressionNames[%s]: %#v \n", k, v)
	}

	scanInput := &dynamodb.ScanInput{
		TableName:                 &d.tableName,
		ExpressionAttributeNames:  expressionNames,
		ExpressionAttributeValues: paramsExpression,
		FilterExpression:          aws.String(filterExpression),
		//ConsistentRead:            aws.Bool(true),
	}

	//if query.After != nil {
	//	schemed, err := schema.FromJSON([]byte(*query.After))
	//	if err != nil {
	//		return PageResult[Record[schema.Schema]]{}, err
	//	}
	//
	//	scanInput.ExclusiveStartKey = map[string]types.AttributeValue{
	//		"ID": &types.AttributeValueMemberS{
	//			Value: schema.AsDefault[string](schema.Get(schemed, "ID"), ""),
	//		},
	//		"Type": &types.AttributeValueMemberS{
	//			Value: schema.AsDefault[string](schema.Get(schemed, "Type"), ""),
	//		},
	//	}
	//}

	// Be aware that DynamoDB limit is scan limit, not page limit!
	if query.Limit > 0 {
		scanInput.Limit = aws.Int32(int32(query.Limit))
	}

	items, err := d.client.Scan(context.Background(), scanInput)
	if err != nil {
		return PageResult[Record[schema.Schema]]{}, err
	}

	result := PageResult[Record[schema.Schema]]{
		Items: nil,
	}

	for _, item := range items.Items {
		// normalize input for further processing
		i := &types.AttributeValueMemberM{
			Value: item,
		}

		schemed, err := schema.FromDynamoDB(i)
		if err != nil {
			return PageResult[Record[schema.Schema]]{}, err
		}

		typed, err := schema.ToGoG[*Record[schema.Schema]](schemed)
		if err != nil {
			return PageResult[Record[schema.Schema]]{}, err
		}
		result.Items = append(result.Items, *typed)
	}

	//if items.LastEvaluatedKey != nil {
	//	after := &types.AttributeValueMemberM{
	//		Value: items.LastEvaluatedKey,
	//	}
	//	schemed, err := schema.FromDynamoDB(after)
	//	if err != nil {
	//		return PageResult[Record[schema.Schema]]{}, fmt.Errorf("DynamoDBRepository.FindingRecords: error calculating after cursor %s. %w", err, ErrInternalError)
	//	}
	//	json, err := schema.ToJSON(schemed)
	//	if err != nil {
	//		return PageResult[Record[schema.Schema]]{}, fmt.Errorf("DynamoDBRepository.FindingRecords: error serializing after cursor %s. %w", err, ErrInternalError)
	//	}
	//	cursor := string(json)
	//	result.Next = &FindingRecords[Record[schema.Schema]]{
	//		Where: query.Where,
	//		Sort:  query.Sort,
	//		Limit: query.Limit,
	//		After: &cursor,
	//	}
	//}

	return result, nil
}

func (d *DynamoDBRepository) buildFilterExpression(query FindingRecords[Record[schema.Schema]]) (string, map[string]types.AttributeValue, map[string]string, error) {
	var where predicate.Predicate
	var binds predicate.ParamBinds = map[predicate.BindName]schema.Schema{}
	var names map[string]string = map[string]string{}

	if query.RecordType != "" {
		names["Type"] = "#Type"
		where = &predicate.Compare{
			Location:  "Type",
			Operation: "=",
			BindValue: &predicate.BindValue{BindName: ":Type"},
		}
		binds[":Type"] = schema.MkString(query.RecordType)
	}

	if query.Where != nil {
		if where == nil {
			where = query.Where.Predicate
			binds = query.Where.Params
		} else {
			where = &predicate.And{
				L: []predicate.Predicate{where, query.Where.Predicate},
			}

			for k, v := range query.Where.Params {
				if _, ok := binds[k]; ok {
					return "", nil, nil, fmt.Errorf("store.DynamoDBRepository.FindingRecords: duplicated bind value: %s", k)
				}

				binds[k] = v
			}
		}
	}

	if where == nil {
		return "", nil, nil, nil
	}

	expression := toExpression(where, names)

	// reverse names
	reverser := map[string]string{}
	for k, v := range names {
		reverser[v] = k
	}

	return expression, toAttributes(binds), reverser, nil
}

func toExpression(where predicate.Predicate, names map[string]string) string {
	return predicate.MustMatchPredicate(
		where,
		func(x *predicate.And) string {
			var result []string
			for _, v := range x.L {
				result = append(result, toExpression(v, names))
			}

			return strings.Join(result, " AND ")
		},
		func(x *predicate.Or) string {
			var result []string
			for _, v := range x.L {
				result = append(result, toExpression(v, names))
			}

			return strings.Join(result, " OR ")

		},
		func(x *predicate.Not) string {
			return "NOT " + toExpression(x.P, names)
		},
		func(x *predicate.Compare) string {
			// Because of https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Expressions.ExpressionAttributeNames.html
			// we need to make sure that all names are not reserved keyword, so we add a counter to the end of the name in case of collision
			var named []string
			//var parts []string = strings.Split(x.Location, ".")

			locs, err := schema.ParseLocation(x.Location)
			if err != nil {
				panic(err)
			}

			for _, loc := range locs {
				part := schema.MustMatchLocation(
					loc,
					func(x *schema.LocationField) string {
						return x.Name
					},
					func(x *schema.LocationIndex) string {
						panic("implement me")
					},
					func(x *schema.LocationAnything) string {
						return "schema.Map"
					},
				)

				name := part
				if strings.Contains(name, ".") {
					name = strings.ReplaceAll(name, ".", "_")
				}

				if _, ok := names[part]; !ok {
					names[part] = "#" + name
				}

				named = append(named, names[part])
			}

			//for _, part := range parts {
			//	if _, ok := names[part]; !ok {
			//		// TODO(schema.Union) find a better way to handle # union separator
			//		// for example insteaf hard coded schema.Map use schema.UnionType....
			//		if part == "[*]" {
			//			part = "schema.Map"
			//			names[part] = "#hash"
			//		} else {
			//			names[part] = "#" + part
			//		}
			//	}
			//	named = append(named, names[part])
			//}

			return predicate.MustMatchBindable(
				x.BindValue,
				func(y *predicate.BindValue) string {
					return strings.Join(named, ".") + " " + x.Operation + " " + y.BindName
				},
				func(y *predicate.Literal) string {
					panic("implement me")
				},
				func(y *predicate.Locatable) string {
					panic("implement me")
				},
			)
		},
	)
}

func toAttributes(binds predicate.ParamBinds) map[string]types.AttributeValue {
	result := map[string]types.AttributeValue{}
	for k, v := range binds {
		result[k] = schema.ToDynamoDB(v)
	}

	return result
}
