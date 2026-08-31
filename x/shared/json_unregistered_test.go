//go:tag mkunion:",no-type-registry"
package shared

import (
	"strings"
	"testing"
)

// unregisteredUnion simulates a mkunion union interface whose generated
// init() registration was never linked into the binary (e.g. package compiled
// with no-type-registry, or the registering package was not imported).
//
// Before the fix, JSONMarshal silently fell back to plain json.Marshal:
// no $type discriminator, no error. The corruption was only discovered
// later, when JSONUnmarshal failed on the written data.
//go:tag shape:"-"
type unregisteredUnion interface {
	acceptUnregistered()
}

//go:tag shape:"-"
type unregisteredVariant struct {
	Name string
}

func (*unregisteredVariant) acceptUnregistered() {}

func TestJSONMarshal_UnregisteredUnion_FailsLoudly(t *testing.T) {
	var value unregisteredUnion = &unregisteredVariant{Name: "bob"}

	data, err := JSONMarshal[unregisteredUnion](value)
	if err == nil {
		t.Fatalf("expected error for unregistered interface, got data: %s", string(data))
	}
	if !strings.Contains(err.Error(), "no registered marshaller") {
		t.Errorf("error should explain the registry problem, got: %v", err)
	}
}

func TestJSONUnmarshal_UnregisteredUnion_FailsLoudly(t *testing.T) {
	_, err := JSONUnmarshal[unregisteredUnion]([]byte(`{"Name":"bob"}`))
	if err == nil {
		t.Fatal("expected error for unregistered interface, got nil")
	}
	if !strings.Contains(err.Error(), "no registered marshaller") {
		t.Errorf("error should explain the registry problem, got: %v", err)
	}
}

func TestJSONUnmarshal_UnregisteredUnion_NullIsStillValid(t *testing.T) {
	result, err := JSONUnmarshal[unregisteredUnion]([]byte(`null`))
	if err != nil {
		t.Fatalf("null must unmarshal to the zero value without error, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for null, got: %#v", result)
	}
}

// Plain structs (non-interface types) must keep working through the
// native fallback path exactly as before.
//
//go:tag shape:"-"
type unregisteredPlainStruct struct {
	Name string
}

func TestJSONMarshal_PlainStruct_StillWorks(t *testing.T) {
	data, err := JSONMarshal[unregisteredPlainStruct](unregisteredPlainStruct{Name: "bob"})
	if err != nil {
		t.Fatalf("plain struct marshal must keep working, got: %v", err)
	}

	back, err := JSONUnmarshal[unregisteredPlainStruct](data)
	if err != nil {
		t.Fatalf("plain struct unmarshal must keep working, got: %v", err)
	}
	if back.Name != "bob" {
		t.Fatalf("round trip mismatch: %#v", back)
	}
}
