package search

type Vector []float64

func (v *Vector) Add(x float64) {
	*v = append(*v, x)
}

func (v *Vector) Dot(y Vector) Vector {
	if len(*v) != len(y) {
		panic("search.Vector.Dot: len(v) != len(y)")
	}

	result := NewVector(len(*v))
	for i := 0; i < len(*v); i++ {
		result[i] = (*v)[i] * y[i]
	}

	return result
}

func (v *Vector) Mean() float64 {
	var sum float64
	for _, x := range *v {
		sum += x
	}

	return sum / float64(len(*v))
}

func NewVector(len int) Vector {
	return make(Vector, len)
}
