// Package specdata holds the record payload types used by the spec suite.
//
// It lives in its own leaf package on purpose: the payloads are used as type
// arguments in the backend packages' tests, so the generated type registries
// of those packages import the payloads' package. If the payloads lived in
// the spec package (which imports schemaless), that would be an import cycle.
package specdata

// Engine is a union stored inside the record payload, so the complex suite
// exercises nested-union serialisation and union-path queries on every backend.
//
//go:tag mkunion:"Engine"
type (
	Petrol struct {
		Brand     string
		Cylinders int
	}
	Electric struct {
		Brand string
		KWh   float64
	}
)

// Vehicle is the "complex" record payload: plain fields, a nested union,
// and a list.
//
//go:tag serde:"json"
type Vehicle struct {
	Name   string
	Wheels int
	Engine Engine
	Tags   []string
}
