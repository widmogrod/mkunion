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
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"strings"
)

func NewDynamoDBRepository[A any](client *dynamodb.Client, tableName string) *DynamoDBRepository[A] {
	return &DynamoDBRepository[A]{
		client:    client,
		tableName: tableName,
	}
}

var _ Repository[any] = (*DynamoDBRepository[any])(nil)

type DynamoDBRepository[A any] struct {
	client    *dynamodb.Client
	tableName string
}

func (d *DynamoDBRepository[A]) Get(key, recordType string) (Record[A], error) {
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
		return Record[A]{}, fmt.Errorf("DynamoDBRepository.GetSchema error=%s. %w", err, ErrInternalError)
	}

	if len(item.Item) == 0 {
		return Record[A]{}, fmt.Errorf("DynamoDBRepository.GetSchema not found. %w", ErrNotFound)
	}

	i := &types.AttributeValueMemberM{
		Value: item.Item,
	}

	schemed, err := schema.FromDynamoDB(i)
	if err != nil {
		return Record[A]{}, fmt.Errorf("DynamoDBRepository.GetSchema schema conversion error=%s. %w", err, ErrInternalError)
	}

	typed, err := schema.ToGoG[*Record[A]](schemed)
	if err != nil {
		return Record[A]{}, fmt.Errorf("DynamoDBRepository.GetSchema type conversion error=%s. %w", err, ErrInvalidType)
	}

	return *typed, nil
}

func (d *DynamoDBRepository[A]) UpdateRecords(command UpdateRecords[Record[A]]) (*UpdateRecordsResult[Record[A]], error) {
	if command.IsEmpty() {
		return nil, fmt.Errorf("DynamoDBRepository.UpdateRecords: empty command %w", ErrEmptyCommand)
	}

	var transact []types.TransactWriteItem
	for _, value := range command.Saving {
		originalVersion := value.Version
		value.Version++
		sch := schema.FromGo[Record[A]](value)
		item := schema.ToDynamoDB(sch)
		if _, ok := item.(*types.AttributeValueMemberM); !ok {
			return nil, fmt.Errorf("DynamoDBRepository.UpdateRecords: unsupported type: %T", item)
		}

		final, ok := item.(*types.AttributeValueMemberM)
		if !ok {
			return nil, fmt.Errorf("DynamoDBRepository.UpdateRecords: expected map as item. %w", ErrInternalError)
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
						return nil, fmt.Errorf("store.DynamoDBRepository.UpdateRecords: %w", ErrVersionConflict)
					}
				}
			}
		}
		return nil, err
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
}

func (d *DynamoDBRepository[A]) buildScanInput(query FindingRecords[Record[A]]) (*dynamodb.ScanInput, error) {
	filterExpression, paramsExpression, expressionNames, err := d.buildFilterExpression(query)
	if err != nil {
		return nil, err
	}

	log.Infof("\nfilterExpression: %#v \n", filterExpression)
	for k, v := range paramsExpression {
		log.Infof("paramsExpression[%s]: %#v \n", k, v)
	}

	for k, v := range expressionNames {
		log.Infof("expressionNames[%s]: %#v \n", k, v)
	}

	scanInput := &dynamodb.ScanInput{
		TableName: &d.tableName,
		//ConsistentRead: aws.Bool(true),
	}

	// DynamoDB rejects an empty FilterExpression (and empty attribute maps)
	// with a ValidationException, so "list all" must send none of them.
	if filterExpression != "" {
		scanInput.FilterExpression = aws.String(filterExpression)
		if len(expressionNames) > 0 {
			scanInput.ExpressionAttributeNames = expressionNames
		}
		if len(paramsExpression) > 0 {
			scanInput.ExpressionAttributeValues = paramsExpression
		}
	}

	if query.After != nil {
		schemed, err := shared.JSONUnmarshal[schema.Schema]([]byte(*query.After))
		if err != nil {
			return nil, fmt.Errorf("dynamodb.FindingRecords: after cursor unmarshal err ;%w", err)
		}

		scanInput.ExclusiveStartKey = map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{
				Value: schema.GetSchemaDefault[string](schemed, "ID", ""),
			},
			"Type": &types.AttributeValueMemberS{
				Value: schema.GetSchemaDefault[string](schemed, "Type", ""),
			},
		}
	}

	// Be aware that DynamoDB limit is scan limit, not page limit!
	if query.Limit > 0 {
		scanInput.Limit = aws.Int32(int32(query.Limit))
	}

	return scanInput, nil
}

func (d *DynamoDBRepository[A]) nextPageQuery(query FindingRecords[Record[A]], lastEvaluatedKey map[string]types.AttributeValue) (*FindingRecords[Record[A]], error) {
	after := &types.AttributeValueMemberM{
		Value: lastEvaluatedKey,
	}

	schemed, err := schema.FromDynamoDB(after)
	if err != nil {
		return nil, fmt.Errorf("DynamoDBRepository.FindingRecords: error calculating after cursor %s. %w", err, ErrInternalError)
	}

	json, err := shared.JSONMarshal[schema.Schema](schemed)
	if err != nil {
		return nil, fmt.Errorf("DynamoDBRepository.FindingRecords: error serializing after cursor %s. %w", err, ErrInternalError)
	}
	cursor := string(json)
	return &FindingRecords[Record[A]]{
		RecordType: query.RecordType,
		Where:      query.Where,
		Sort:       query.Sort,
		Limit:      query.Limit,
		After:      &cursor,
	}, nil
}

func (d *DynamoDBRepository[A]) FindingRecords(query FindingRecords[Record[A]]) (PageResult[Record[A]], error) {
	scanInput, err := d.buildScanInput(query)
	if err != nil {
		return PageResult[Record[A]]{}, err
	}

	items, err := d.client.Scan(context.Background(), scanInput)
	if err != nil {
		return PageResult[Record[A]]{}, err
	}

	result := PageResult[Record[A]]{
		Items: nil,
	}

	for _, item := range items.Items {
		// normalize input for further processing
		i := &types.AttributeValueMemberM{
			Value: item,
		}

		schemed, err := schema.FromDynamoDB(i)
		if err != nil {
			return PageResult[Record[A]]{}, fmt.Errorf("DynamoDBRepository.FindingRecords: error converting item %s. %w", err, ErrInternalError)
		}

		typed, err := schema.ToGoG[*Record[A]](schemed)
		if err != nil {
			return PageResult[Record[A]]{}, fmt.Errorf("DynamoDBRepository.FindingRecords: error converting item %s. %w", err, ErrInternalError)
		}
		result.Items = append(result.Items, *typed)
	}

	if items.LastEvaluatedKey != nil {
		next, err := d.nextPageQuery(query, items.LastEvaluatedKey)
		if err != nil {
			return PageResult[Record[A]]{}, err
		}
		result.Next = next
	}

	return result, nil
}

func (d *DynamoDBRepository[A]) buildFilterExpression(query FindingRecords[Record[A]]) (string, map[string]types.AttributeValue, map[string]string, error) {
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
			for k, v := range query.Where.Params {
				binds[k] = v
			}
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

	builder := &dynamoDBExpressionBuilder{names: names, binds: binds}
	expression, err := builder.toExpression(where)
	if err != nil {
		return "", nil, nil, fmt.Errorf("store.DynamoDBRepository.FindingRecords: %w", err)
	}

	// reverse names
	reverser := map[string]string{}
	for k, v := range names {
		reverser[v] = k
	}

	return expression, toAttributes(builder.binds), reverser, nil
}

type dynamoDBExpressionBuilder struct {
	names      map[string]string
	binds      predicate.ParamBinds
	litCounter int
}

func (b *dynamoDBExpressionBuilder) toExpression(where predicate.Predicate) (string, error) {
	return predicate.MatchPredicateR2(
		where,
		func(x *predicate.And) (string, error) {
			result, err := b.groupExpressions(x.L)
			if err != nil {
				return "", err
			}

			return strings.Join(result, " AND "), nil
		},
		func(x *predicate.Or) (string, error) {
			result, err := b.groupExpressions(x.L)
			if err != nil {
				return "", err
			}

			return strings.Join(result, " OR "), nil
		},
		func(x *predicate.Not) (string, error) {
			expr, err := b.groupExpression(x.P)
			if err != nil {
				return "", err
			}

			return "NOT " + expr, nil
		},
		func(x *predicate.Compare) (string, error) {
			left, err := b.locationToPath(x.Location)
			if err != nil {
				return "", err
			}

			return predicate.MatchBindableR2(
				x.BindValue,
				func(y *predicate.BindValue) (string, error) {
					return left + " " + x.Operation + " " + y.BindName, nil
				},
				func(y *predicate.Literal) (string, error) {
					bindName := fmt.Sprintf(":lit%d", b.litCounter)
					b.litCounter++
					b.binds[bindName] = y.Value
					return left + " " + x.Operation + " " + bindName, nil
				},
				func(y *predicate.Locatable) (string, error) {
					right, err := b.locationToPath(y.Location)
					if err != nil {
						return "", err
					}
					return left + " " + x.Operation + " " + right, nil
				},
			)
		},
	)
}

func (b *dynamoDBExpressionBuilder) groupExpressions(xs []predicate.Predicate) ([]string, error) {
	var result []string
	for _, v := range xs {
		expr, err := b.groupExpression(v)
		if err != nil {
			return nil, err
		}
		result = append(result, expr)
	}

	return result, nil
}

// groupExpression parenthesises And/Or sub-expressions, so that operator
// precedence in DynamoDB (NOT > AND > OR) cannot regroup them,
// i.e. Not{Or{a,b}} renders as `NOT (a OR b)`, never `NOT a OR b`.
func (b *dynamoDBExpressionBuilder) groupExpression(p predicate.Predicate) (string, error) {
	expr, err := b.toExpression(p)
	if err != nil {
		return "", err
	}

	needsParens := false
	switch x := p.(type) {
	case *predicate.And:
		needsParens = len(x.L) > 1
	case *predicate.Or:
		needsParens = len(x.L) > 1
	}

	if needsParens {
		return "(" + expr + ")", nil
	}

	return expr, nil
}

func (b *dynamoDBExpressionBuilder) locationToPath(location string) (string, error) {
	locs, err := schema.ParseLocation(location)
	if err != nil {
		return "", fmt.Errorf("dynamodb.locationToPath: parse location %q: %w", location, err)
	}

	var path strings.Builder
	for _, loc := range locs {
		part, err := schema.MatchLocationR2(
			loc,
			func(x *schema.LocationField) (string, error) {
				return b.nameAlias(x.Name), nil
			},
			func(x *schema.LocationIndex) (string, error) {
				return fmt.Sprintf("[%d]", x.Index), nil
			},
			func(x *schema.LocationAnything) (string, error) {
				// TODO(schema.union) find a better way to handle # union separator
				// for example instead of hard coded schema.Map use schema.UnionType....
				return b.nameAlias("schema.Map"), nil
			},
		)
		if err != nil {
			return "", err
		}

		if path.Len() > 0 && !strings.HasPrefix(part, "[") {
			path.WriteString(".")
		}
		path.WriteString(part)
	}

	return path.String(), nil
}

// nameAlias maps an attribute name to an expression alias (#name), because
// attribute names can collide with DynamoDB reserved keywords
// https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Expressions.ExpressionAttributeNames.html
func (b *dynamoDBExpressionBuilder) nameAlias(part string) string {
	if _, ok := b.names[part]; !ok {
		b.names[part] = "#" + strings.ReplaceAll(part, ".", "_")
	}

	return b.names[part]
}

func toAttributes(binds predicate.ParamBinds) map[string]types.AttributeValue {
	result := map[string]types.AttributeValue{}
	for k, v := range binds {
		result[k] = schema.ToDynamoDB(v)
	}

	return result
}
