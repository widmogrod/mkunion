package shared

import (
	"encoding/json"
	"fmt"
)

func JSONParseObject(x []byte, onElement func(key string, value []byte) error) error {
	var jsonMap map[string]json.RawMessage
	err := json.Unmarshal(x, &jsonMap)
	if err != nil {
		return err
	}

	for key, value := range jsonMap {
		if err := onElement(key, value); err != nil {
			return fmt.Errorf("shared.JSONParseObject: onElement() for key=%s; %w", key, err)
		}
	}

	return nil
}

func JSONToListWithDeserializer[A any](x []byte, _ []A, deserialize func(value []byte) (A, error)) ([]A, error) {
	var jsonList []json.RawMessage
	err := json.Unmarshal(x, &jsonList)
	if err != nil {
		return nil, fmt.Errorf("shared.JSONToListWithDeserializer: %w", err)
	}

	var result []A
	for idx, value := range jsonList {
		out, err := deserialize(value)
		if err != nil {
			return nil, fmt.Errorf("shared.JSONToListWithDeserializer: deserialize() for index=%d; %w", idx, err)
		}

		result = append(result, out)
	}

	return result, nil
}

func JSONToMapWithDeserializer[K comparable, A any](
	x []byte,
	_ map[K]A,
	deserialize func(value []byte) (A, error),
) (map[K]A, error) {
	var jsonMap map[K]json.RawMessage
	err := json.Unmarshal(x, &jsonMap)
	if err != nil {
		return nil, fmt.Errorf("shared.JSONToMapWithDeserializer: %w", err)
	}

	var result = make(map[K]A)
	for key, value := range jsonMap {
		out, err := deserialize(value)
		if err != nil {
			return nil, fmt.Errorf("shared.JSONToMapWithDeserializer: deserialize() for key=%v; %w", key, err)
		}

		result[key] = out
	}

	return result, nil
}

func JSONListFromSerializer[A any](x []A, serialize func(x A) ([]byte, error)) ([]byte, error) {
	var result []json.RawMessage
	for _, value := range x {
		out, err := serialize(value)
		if err != nil {
			return nil, fmt.Errorf("shared.JSONListFromSerializer: serialize() for value=%v; %w", value, err)
		}

		result = append(result, out)
	}

	return json.Marshal(result)
}

func JSONMapFromSerializer[K comparable, A any](
	x map[K]A,
	serialize func(x A) ([]byte, error),
) ([]byte, error) {
	var result = make(map[K][]byte)
	for key, value := range x {
		out, err := serialize(value)
		if err != nil {
			return nil, fmt.Errorf("shared.JSONMapFromSerializer: serialize() for key=%v; %w", key, err)
		}

		result[key] = out
	}

	return json.Marshal(result)
}
