package projection

import "github.com/widmogrod/mkunion/x/schema"

type CountHandler struct {
	value int
}

func (h *CountHandler) Process(msg Item, returning func(Item)) error {
	h.value += schema.AsDefault[int](msg.Data, 0)
	returning(Item{
		Data: schema.MkInt(int64(h.value)),
	})
	return nil
}
