package persistent

import (
	"fmt"
	"reflect"
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

type Config[C, S any] struct {
	createCommands map[reflect.Type]bool
	updateCommands map[reflect.Type]func(any) *predicate.WherePredicates
	stateIDFunc    func(S) (string, bool)
}

func Configure[C, S any]() *Config[C, S] {
	return &Config[C, S]{
		createCommands: make(map[reflect.Type]bool),
		updateCommands: make(map[reflect.Type]func(any) *predicate.WherePredicates),
	}
}

func (c *Config[C, S]) CommandThatCreatesState(examples ...C) *Config[C, S] {
	for _, example := range examples {
		c.createCommands[reflect.TypeOf(example)] = true
	}
	return c
}

func (c *Config[C, S]) CommandThatUpdatesState(example C, queryFunc func(C) *predicate.WherePredicates) *Config[C, S] {
	cmdType := reflect.TypeOf(example)
	c.updateCommands[cmdType] = func(cmd any) *predicate.WherePredicates {
		return queryFunc(cmd.(C))
	}
	return c
}

func (c *Config[C, S]) StateIDFrom(f func(S) (string, bool)) *Config[C, S] {
	c.stateIDFunc = f
	return c
}

func (c *Config[C, S]) IsCreateCommand(cmd C) bool {
	return c.createCommands[reflect.TypeOf(cmd)]
}

func (c *Config[C, S]) IsUpdateCommand(cmd C) bool {
	_, ok := c.updateCommands[reflect.TypeOf(cmd)]
	return ok
}

func (c *Config[C, S]) GetUpdateQuery(cmd C) *predicate.WherePredicates {
	if queryFunc, ok := c.updateCommands[reflect.TypeOf(cmd)]; ok {
		return queryFunc(cmd)
	}
	return nil
}

func (c *Config[C, S]) Validate() error {
	if c.stateIDFunc == nil {
		return fmt.Errorf("StateIDFrom must be configured")
	}
	return nil
}

func ExtractStateIDFromLocation[S any](location string) func(S) (string, bool) {
	return func(state S) (string, bool) {
		// Use reflection to get the actual value
		val := reflect.ValueOf(state)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		
		// Look for the field by name
		field := val.FieldByName(location)
		if !field.IsValid() {
			return "", false
		}
		
		// Check if it's a string
		if field.Kind() == reflect.String {
			return field.String(), true
		}
		
		return "", false
	}
}

func ExtractStateIDUsingShape[S any]() func(S) (string, bool) {
	commonIDFields := []string{"ID", "Id", "id"}
	
	return func(state S) (string, bool) {
		// Use reflection to get the actual value
		val := reflect.ValueOf(state)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		
		// Try common field names
		for _, fieldName := range commonIDFields {
			field := val.FieldByName(fieldName)
			if field.IsValid() && field.Kind() == reflect.String {
				return field.String(), true
			}
		}
		
		return "", false
	}
}