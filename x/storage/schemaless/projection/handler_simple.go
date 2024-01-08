package projection

var _ Handler = (*SimpleProcessHandler)(nil)

type SimpleProcessHandler struct {
	P func(x Item, returning func(Item)) error
	R func(x Item, returning func(Item)) error
}

func (s *SimpleProcessHandler) Process(x Item, returning func(Item)) error {
	return s.P(x, returning)
}

func (s *SimpleProcessHandler) Retract(x Item, returning func(Item)) error {
	//TODO implement me
	panic("implement me")
}

//var _ HandleAccumulate = (*SimpleAccumulateHandler)(nil)
//
//type SimpleAccumulateHandler struct {
//	P func(current Item, previous *Item, returning func(Item)) error
//}
//
//func (s *SimpleAccumulateHandler) ProcessAccumulate(current Item, previous *Item, returning func(Item)) error {
//	return s.P(current, previous, returning)
//}
//
//type SimpleAccumulateAndRetractHandler struct {
//}
//
//func (s *SimpleAccumulateAndRetractHandler) ProcessAccumulateAndRetract(current Item, retract *Item, returning func(Item)) error {
//	//TODO implement me
//	panic("implement me")
//}
//
//var _ HandleAccumulateAndRetract = (*SimpleAccumulateAndRetractHandler)(nil)
