package shape

import (
	"encoding/json"
)

func JsonParseObject(x []byte, f func(key string, value []byte) error) error {
	var data map[string]json.RawMessage
	err := json.Unmarshal(x, &data)
	if err != nil {
		return err
	}

	for key, value := range data {
		if err := f(key, value); err != nil {
			return err
		}
	}

	return nil
}
