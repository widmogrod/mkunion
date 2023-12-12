package schema

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func FromJSON(data []byte) (Schema, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var x any
	if err := json.Unmarshal(data, &x); err != nil {
		return nil, err
	}

	return FromGo(x), nil
}

func ToJSON(schema Schema) ([]byte, error) {
	res := bytes.Buffer{}
	err := toJSON(schema, &res)
	if err != nil {
		return nil, err
	}
	return res.Bytes(), nil
}

func toJSON(schema Schema, res *bytes.Buffer) error {
	return MustMatchSchema(
		schema,
		func(x *None) error {
			res.WriteString("null")
			return nil
		},
		func(x *Bool) error {
			if *x {
				res.WriteString("true")
			} else {
				res.WriteString("false")
			}
			return nil
		},
		func(x *Number) error {
			_, err := fmt.Fprintf(res, "%f", *x)
			if err != nil {
				return err
			}
			return nil
		},
		func(x *String) error {
			_, err := fmt.Fprintf(res, "%q", *x)
			if err != nil {
				return err
			}
			return nil

		},
		func(x *Binary) error {
			_, err := fmt.Fprintf(res, "%q", base64.StdEncoding.EncodeToString(x.B))
			if err != nil {
				return err
			}
			return nil
		},
		func(x *List) error {
			res.WriteString("[")
			for i, item := range *x {
				if i > 0 {
					res.WriteString(",")
				}
				err := toJSON(item, res)
				if err != nil {
					return err
				}
			}
			res.WriteString("]")
			return nil

		},
		func(x *Map) error {
			res.WriteString("{")
			var i int
			for key, value := range *x {
				if i > 0 {
					res.WriteString(",")
				}
				i++

				_, err := fmt.Fprintf(res, "%q:", key)
				if err != nil {
					return err
				}
				err = toJSON(value, res)
				if err != nil {
					return err
				}
			}
			res.WriteString("}")
			return nil
		},
	)
}
