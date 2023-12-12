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
		func(x *Binary) types.AttributeValue {
			return &types.AttributeValueMemberB{
				Value: x.B,
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
			for key, value := range *x {
				result.Value[key] = ToDynamoDB(value)
			}
			return result
		},
	)
}

func FromDynamoDB(x types.AttributeValue) (Schema, error) {
	switch y := x.(type) {
	case *types.AttributeValueMemberB:
		return &Binary{B: y.Value}, nil

	case *types.AttributeValueMemberBS:
		result := &List{
			Items: []Schema{},
		}
		for _, item := range y.Value {
			result.Items = append(result.Items, &Binary{B: item})
		}
		return result, nil

	case *types.AttributeValueMemberNS:
		result := &List{
			Items: []Schema{},
		}
		for _, item := range y.Value {
			num, err := strconv.ParseFloat(item, 64)
			if err != nil {
				return nil, err
			}

			result.Items = append(result.Items, MkFloat(num))
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
		return MkBool(y.Value), nil

	case *types.AttributeValueMemberN:
		num, err := strconv.ParseFloat(y.Value, 64)
		if err != nil {
			return nil, err
		}

		return MkFloat(num), nil

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
		result := make(Map)
		for name, item := range y.Value {
			v, err := FromDynamoDB(item)
			if err != nil {
				return nil, err
			}

			result[name] = v
		}

		return &result, nil
	}

	panic("unreachable")
}

func UnwrapDynamoDB(data Schema) (Schema, error) {
	switch x := data.(type) {
	case *Map:
		if len(*x) == 1 {
			for name, value := range *x {
				switch name {
				case "S":
					value := AsDefault[string](value, "")
					return FromDynamoDB(&types.AttributeValueMemberS{
						Value: value,
					})
				case "SS":
					switch y := value.(type) {
					case *List:
						result := &List{}
						for _, item := range y.Items {
							result.Items = append(result.Items, MkString(AsDefault[string](item, "")))
						}
						return result, nil
					default:
						return nil, fmt.Errorf("schema.UnwrapDynamoDB: unknown type (2): %s=%T", name, value)
					}
				case "N":
					value := AsDefault[string](value, "")
					return FromDynamoDB(&types.AttributeValueMemberN{
						Value: value,
					})
				case "NS":
					switch y := value.(type) {
					case *List:
						result := &List{}
						for _, item := range y.Items {
							result.Items = append(result.Items, MkFloat(AsDefault[float64](item, 0)))
						}
						return result, nil
					default:
						return nil, fmt.Errorf("schema.UnwrapDynamoDB: unknown type (2): %s=%T", name, value)
					}
				case "B":
					// Assumption is that here we have base64 encoded string from DynamoDB
					// and not a binary value, so to not do double encoding we just
					// pas it as is. This assumption, makse only sence, when it's used on values that
					// require unwrapping DynamoDB format.
					//Which may imply, that those values are ie from other medium than DynamoDB.
					return value, nil

				case "BS":
					switch y := value.(type) {
					case *List:
						result := &List{}
						for _, item := range y.Items {
							result.Items = append(result.Items, item)
						}
						return result, nil
					default:
						return nil, fmt.Errorf("schema.UnwrapDynamoDB: unknown type (2): %s=%T", name, value)
					}

				case "BOOL":
					value := AsDefault[bool](value, false)
					return FromDynamoDB(&types.AttributeValueMemberBOOL{
						Value: value,
					})
				case "NULL":
					return &None{}, nil

				case "M":
					switch y := value.(type) {
					case *Map:
						return assumeMap(y)
					default:
						return nil, fmt.Errorf("schema.UnwrapDynamoDB: unknown type (1): %s=%T", name, value)
					}

				case "L":
					switch y := value.(type) {
					case *List:
						result := &List{}
						for _, item := range y.Items {
							unwrapped, err := UnwrapDynamoDB(item)
							if err != nil {
								return nil, err
							}
							result.Items = append(result.Items, unwrapped)
						}
						return result, nil
					default:
						return nil, fmt.Errorf("schema.UnwrapDynamoDB: unknown type (2): %s=%T", name, value)
					}

				default:
					return nil, fmt.Errorf("schema.UnwrapDynamoDB: unknown type (3): %s=%T", name, value)
				}
			}
		} else {
			return assumeMap(x)
		}
	}

	return nil, fmt.Errorf("schema.UnwrapDynamoDB: unknown type (4): %T", data)
}

func assumeMap(x *Map) (Schema, error) {
	result := make(Map)
	for key, value := range *x {
		value, err := UnwrapDynamoDB(value)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return &result, nil
}
