package schema

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"strconv"
)

func ToDynamoDB(x Schema) types.AttributeValue {
	return MustMatchSchema(
		x,
		func(x *None) types.AttributeValue {
			return &types.AttributeValueMemberNULL{
				Value: true,
			}
		},
		func(x *Bool) types.AttributeValue {
			return &types.AttributeValueMemberBOOL{
				Value: bool(*x),
			}
		},
		func(x *Number) types.AttributeValue {
			return &types.AttributeValueMemberN{
				Value: fmt.Sprintf("%f", *x),
			}
		},
		func(x *String) types.AttributeValue {
			return &types.AttributeValueMemberS{
				Value: string(*x),
			}
		},
		func(x *List) types.AttributeValue {
			result := &types.AttributeValueMemberL{
				Value: []types.AttributeValue{},
			}
			for _, item := range x.Items {
				result.Value = append(result.Value, ToDynamoDB(item))
			}
			return result
		},
		func(x *Map) types.AttributeValue {
			result := &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{},
			}
			for _, item := range x.Field {
				result.Value[item.Name] = ToDynamoDB(item.Value)
			}
			return result
		},
	)
}

func FromDynamoDB(x types.AttributeValue) (Schema, error) {
	switch y := x.(type) {
	case *types.AttributeValueMemberB:
		return nil, fmt.Errorf("FromDynamoDB: unsupported type: %T", x)

	case *types.AttributeValueMemberBS:
		return nil, fmt.Errorf("FromDynamoDB: unsupported type: %T", x)

	case *types.AttributeValueMemberNS:
		result := &List{
			Items: []Schema{},
		}
		for _, item := range y.Value {
			num, err := strconv.ParseFloat(item, 64)
			if err != nil {
				return nil, err
			}

			v := Number(num)
			result.Items = append(result.Items, &v)
		}
		return result, nil

	case *types.AttributeValueMemberSS:
		result := &List{
			Items: []Schema{},
		}
		for _, item := range y.Value {
			result.Items = append(result.Items, MkString(item))
		}
		return result, nil

	case *types.AttributeValueMemberNULL:
		return &None{}, nil

	case *types.AttributeValueMemberBOOL:
		v := Bool(y.Value)
		return &v, nil

	case *types.AttributeValueMemberN:
		num, err := strconv.ParseFloat(y.Value, 64)
		if err != nil {
			return nil, err
		}

		v := Number(num)
		return &v, nil

	case *types.AttributeValueMemberS:
		return MkString(y.Value), nil

	case *types.AttributeValueMemberL:
		result := &List{
			Items: []Schema{},
		}
		for _, item := range y.Value {
			v, err := FromDynamoDB(item)
			if err != nil {
				return nil, err
			}
			result.Items = append(result.Items, v)
		}
		return result, nil

	case *types.AttributeValueMemberM:
		result := &Map{
			Field: []Field{},
		}
		for name, item := range y.Value {
			v, err := FromDynamoDB(item)
			if err != nil {
				return nil, err
			}

			result.Field = append(result.Field, Field{
				Name:  name,
				Value: v,
			})
		}

		return result, nil
	}

	panic("unreachable")
}
