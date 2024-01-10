package stream

import "fmt"

//go:generate go run ../../cmd/mkunion

var (
	ErrEndOfStream     = fmt.Errorf("end of stream")
	ErrOffsetSetOnPush = fmt.Errorf("offset set on push")
)

//go:tag mkunion:"PullCMD"
type (
	FromBeginning struct{}
	FromOffset    = Offset
)

//go:tag serde:"json"
type Offset string

func (o *Offset) IsSet() bool {
	return o != nil && *o != ""
}

type Item[A any] struct {
	Key    string
	Data   A
	Offset *Offset
}

type Stream[A any] interface {
	Push(x *Item[A]) error
	Pull(offset PullCMD) (*Item[A], error)
}

var (
	ErrEmptyOffsetInParsing = fmt.Errorf("offset fail; empty offset")
	ErrParsingOffset        = fmt.Errorf("offset fail; parsing offset")
)

func MkOffsetFromInt(x int) *Offset {
	result := Offset(fmt.Sprintf("%d", x))
	return &result
}

func ParseOffsetAsInt(x *Offset) (int, error) {
	if x == nil {
		return 0, ErrEmptyOffsetInParsing
	}

	var result int

	_, err := fmt.Sscanf(string(*x), "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("stream.ParseOffsetAsInt: %w; %w", err, ErrParsingOffset)
	}

	return result, nil
}
