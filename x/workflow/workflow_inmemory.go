package workflow

var _ Dependency = (*DI)(nil)

type DI struct {
	FindFunctionF       func(funcID string) (Function, error)
	FindWorkflowF       func(flowID string) (*Flow, error)
	GenerateCallbackIDF func() string
}

func (di *DI) FindWorkflow(flowID string) (*Flow, error) {
	return di.FindWorkflowF(flowID)
}

func (di *DI) FindFunction(funcID string) (Function, error) {
	return di.FindFunctionF(funcID)
}

func (di *DI) GenerateCallbackID() string {
	return di.GenerateCallbackIDF()
}
