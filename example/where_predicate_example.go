package example

//go:generate go run ../cmd/mkunion/main.go -name=WherePredicate -skip-extension=schema
type (
	Eq   struct{ V interface{} }
	And  []WherePredicate
	Or   []WherePredicate
	Path struct {
		Parts     []string
		Condition WherePredicate
		Then      []WherePredicate
		X         Eq
		Y         map[string]WherePredicate
	}
)
