package projection

//type FunctionID string
//
//var (
//	ErrFunctionNotFound = fmt.Errorf("function not found")
//	ErrFunctionExists   = fmt.Errorf("function already exists")
//)
//
//type (
//	FunctionRegistry struct {
//		function map[FunctionID]Handler
//	}
//)
//
//func (r *FunctionRegistry) Get(id FunctionID) (Handler, error) {
//	if h, ok := r.function[id]; ok {
//		return h, nil
//	}
//	return nil, fmt.Errorf("%w id=%s", ErrFunctionNotFound, id)
//}
//
//var defaultFunctionRegistry = &FunctionRegistry{
//	function: map[FunctionID]Handler{},
//}
//
//func DefaultFunctionRegistry() *FunctionRegistry {
//	return defaultFunctionRegistry
//}
//
//func WithFunction(id FunctionID, handler Handler) (FunctionID, error) {
//	if _, ok := defaultFunctionRegistry.function[id]; ok {
//		return "", fmt.Errorf("%w id=%s", ErrFunctionExists, id)
//	}
//	defaultFunctionRegistry.function[id] = handler
//	return id, nil
//}
//
//func MustFunction(id FunctionID, handler Handler) FunctionID {
//	res, err := WithFunction(id, handler)
//	if err != nil {
//		panic(err)
//	}
//	return res
//}
//
//func MustRetrieveFunction(id FunctionID) Handler {
//	res, err := defaultFunctionRegistry.Get(id)
//	if err != nil {
//		panic(err)
//	}
//	return res
//}
