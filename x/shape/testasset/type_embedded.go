package testasset

import (
	"time"

	"github.com/widmogrod/mkunion/f"
)

// Embedded shows every way a field can be embedded in a struct.
type Embedded struct {
	ListOf[int]
	f.Some[time.Time]
	*time.Time
	ListOf2[string, int]
	Name string
}
