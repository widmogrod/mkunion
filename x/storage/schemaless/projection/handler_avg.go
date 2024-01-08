package projection

import "github.com/widmogrod/mkunion/x/schema"

type AvgHandler struct {
	avg   float64
	count int
}

func (h *AvgHandler) Process(msg Item, returning func(Item)) error {
	h.avg = (h.avg*float64(h.count) + schema.AsDefault[float64](msg.Data, 0)) / (float64(h.count) + 1)
	// avg = (avg * count + x) / (count + 1)
	h.count += 1

	newValue := schema.Number(h.avg)

	returning(Item{
		Data: &newValue,
	})
	return nil
}
