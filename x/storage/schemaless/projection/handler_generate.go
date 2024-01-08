package projection

var _ Handler = &GenerateHandler{}

type GenerateHandler struct {
	Load func(push func(message Item)) error
}

func (h *GenerateHandler) Process(x Item, returning func(Item)) error {
	return h.Load(returning)
}

func (h *GenerateHandler) Retract(x Item, returning func(Item)) error {
	//TODO implement me
	panic("implement me")
}
