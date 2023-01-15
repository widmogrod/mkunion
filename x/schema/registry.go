package schema

var defaultRegistry *Registry

func init() {
	defaultRegistry = NewRegistry()
}

func RegisterTransformations(xs []TransformFunc) {
	defaultRegistry.RegisterTransformations(xs)
}

func RegisterRules(xs []RuleMatcher) {
	defaultRegistry.RegisterRules(xs)
}

func NewRegistry() *Registry {
	return &Registry{
		transformations: nil,
		matchingRules:   nil,
	}
}

type Registry struct {
	transformations []TransformFunc
	matchingRules   []RuleMatcher
}

func (r *Registry) RegisterTransformations(xs []TransformFunc) {
	r.transformations = append(r.transformations, xs...)
}

func (r *Registry) RegisterRules(xs []RuleMatcher) {
	r.matchingRules = append(r.matchingRules, xs...)
}
