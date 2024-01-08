package projection

import "time"

var _ Handler = &DebounceHandler{}

type DebounceHandler struct {
	MaxSize int
	MaxTime time.Duration

	last     *Item
	lastTime *time.Time
	lastSize int
}

func (t *DebounceHandler) Process(x Item, returning func(Item)) error {
	panic("TODO figure out how to implement timeouts, so that runtime can stimulate a handler to flush its buffer")

	//if t.last == nil {
	//	t.lastTime = new(time.Time)
	//	*t.lastTime = time.Now()
	//}
	//
	//t.last = &x
	//t.lastSize += 1
	//
	//// flush because size limit was reached
	//if t.lastSize >= t.MaxSize && t.MaxSize != 0 {
	//	returning(*t.last)
	//	t.last = nil
	//	t.lastSize = 0
	//	t.lastTime = nil
	//	return nil
	//}
	//
	//// flush because time limit was reached
	//if t.lastTime != nil && time.Since(*t.lastTime) >= t.MaxTime {
	//	returning(*t.last)
	//	t.last = nil
	//	t.lastSize = 0
	//	t.lastTime = nil
	//	return nil
	//}

	return nil
}

func (t *DebounceHandler) Retract(x Item, returning func(Item)) error {
	//TODO implement me
	panic("implement me")
}
