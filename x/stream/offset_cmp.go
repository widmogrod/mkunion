package stream

import "fmt"

var offsetCompare = map[string]func(Offset, Offset) (int8, error){}

func RegisterOffsetCompare(x string, y func(Offset, Offset) (int8, error)) {
	if _, ok := offsetCompare[x]; ok {
		panic(fmt.Errorf("stream.RegisterOffsetCompare: already registered offset namespace '%s'", x))
	}

	offsetCompare[x] = y
}

func OffsetCompare(a, b Offset) (int8, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, fmt.Errorf("stream.OffsetCompare: empty offset")
	}

	if a[0] != b[0] {
		return 0, fmt.Errorf("stream.OffsetCompare: offset namespace mismatch '%v' != '%v'", a[0], b[0])
	}

	if f, ok := offsetCompare[string(a[0])]; ok {
		res, err := f(a, b)
		if err != nil {
			return 0, fmt.Errorf("stream.OffsetCompare: %w", err)
		}

		return res, nil
	}

	return 0, fmt.Errorf("stream.OffsetCompare: unknown offset namespace '%v'", a[0])
}
