package projection

import (
	"errors"
	"github.com/widmogrod/mkunion/x/schema"
	"math"
	"time"
)

var ErrNotFound = errors.New("node not found")

//go:generate go run ../../../../cmd/mkunion/main.go
//go:generate go run ../../../../cmd/mkunion/main.go serde

//go:tag mkunion:"Node"
type (
	DoWindow struct {
		Ctx   *DefaultContext
		Input Node
	}
	// DoMap implicitly means, merge by key
	DoMap struct {
		Ctx   *DefaultContext
		OnMap Handler
		Input Node
	}
	DoLoad struct {
		Ctx    *DefaultContext
		OnLoad Handler
	}
	DoJoin struct {
		Ctx   *DefaultContext
		Input []Node
	}
)

func GetCtx(node Node) *DefaultContext {
	return MatchNodeR1(
		node,
		func(node *DoWindow) *DefaultContext { return node.Ctx },
		func(node *DoMap) *DefaultContext { return node.Ctx },
		func(node *DoLoad) *DefaultContext { return node.Ctx },
		func(node *DoJoin) *DefaultContext { return node.Ctx },
	)
}

func NodeToString(node Node) string {
	return MatchNodeR1(
		node,
		func(node *DoWindow) string { return "Window" },
		func(node *DoMap) string { return "DoWindow" },
		func(node *DoLoad) string { return "DoLoad" },
		func(node *DoJoin) string { return "DoJoin" },
	)
}

//go:tag serde:"json"
type EventTime = int64

//go:tag serde:"json"
type Window struct {
	Start int64
	End   int64
}

//go:tag serde:"json"
type ItemType uint8

//func (i ItemType) MarshalSchema() (*schema.Map, error) {
//	return schema.MkMap(schema.MkField("itemType", schema.MkInt(uint64(i)))), nil
//}

const (
	ItemAggregation ItemType = iota
	ItemRetractAndAggregate
)

//go:tag serde:"json"
type Item struct {
	Key       string
	Data      schema.Schema
	EventTime EventTime
	Window    *Window
	Type      ItemType
}

//go:tag serde:"json"
type ItemGroupedByKey struct {
	Key  string
	Data []Item
}

//go:tag serde:"json"
type ItemGroupedByWindow struct {
	Key    string
	Data   *schema.List
	Window *Window
}

func PackRetractAndAggregate(x, y schema.Schema) *schema.Map {
	return schema.MkMap(
		schema.MkField("Retract", x),
		schema.MkField("Aggregate", y),
	)
}

//func UnpackRetractAndAggregate(x *schema.Map) (retract schema.Schema, aggregate schema.Schema) {
//	return schema.Get[schema.Schema](x, "Retract"), schema.Get[schema.Schema](x, "Aggregate")
//}

type Handler interface {
	Process(x Item, returning func(Item)) error
	Retract(x Item, returning func(Item)) error
}

//type HandleAccumulate interface {
//	ProcessAccumulate(current Item, previous *Item, returning func(Item)) error
//}
//
//type HandleAccumulateAndRetract interface {
//	ProcessAccumulateAndRetract(current Item, retract *Item, returning func(Item)) error
//}

type Builder interface {
	Load(f Handler, opts ...ContextOptionFunc) Builder
	Window(opts ...ContextOptionFunc) Builder
	Map(f Handler, opts ...ContextOptionFunc) Builder
	Join(a, b Builder, opts ...ContextOptionFunc) Builder
	Build() []Node
}

type ContextOptionFunc func(c *DefaultContext)

func NewContextBuilder(builders ...func(config *DefaultContext)) *DefaultContext {
	config := &DefaultContext{
		wd: &FixedWindow{
			Width: math.MaxInt64,
		},
		td: &AtWatermark{},
		fm: &Discard{},
	}
	for _, builder := range builders {
		builder(config)
	}

	return config
}

func WithWindowDescription(wd WindowDescription) ContextOptionFunc {
	return func(config *DefaultContext) {
		config.wd = wd
	}
}

func WithFixedWindow(width time.Duration) ContextOptionFunc {
	return WithWindowDescription(&FixedWindow{
		Width: width,
	})
}
func WithSlidingWindow(width time.Duration, period time.Duration) ContextOptionFunc {
	return WithWindowDescription(&SlidingWindow{
		Width:  width,
		Period: period,
	})
}
func WithSessionWindow(gap time.Duration) ContextOptionFunc {
	return WithWindowDescription(&SessionWindow{
		GapDuration: gap,
	})
}

func WithTriggers(and ...TriggerDescription) ContextOptionFunc {
	return func(config *DefaultContext) {
		config.td = &AllOf{
			Triggers: and,
		}
	}
}

func WithWindowFlushMode(fm WindowFlushMode) ContextOptionFunc {
	return func(config *DefaultContext) {
		config.fm = fm
	}
}

func WithDiscard() ContextOptionFunc {
	return WithWindowFlushMode(&Discard{})
}
func WithAccumulate() ContextOptionFunc {
	return WithWindowFlushMode(&Accumulate{})
}
func WithAccumulatingAndRetracting() ContextOptionFunc {
	return WithWindowFlushMode(&AccumulatingAndRetracting{})
}

func WithName(name string) ContextOptionFunc {
	return func(c *DefaultContext) {
		c.name = name
	}
}

type DefaultContext struct {
	name        string
	contextName string
	//retracting  *bool

	wd WindowDescription
	td TriggerDescription
	fm WindowFlushMode
}

func (c *DefaultContext) Scope(name string) *DefaultContext {
	return NewContextBuilder(WithName(c.name + "." + name))
}

func (c *DefaultContext) Name() string {
	return c.name
}

//go:tag serde:"json"
type Message struct {
	Offset int
	// at some point of time i may need to pass type reference
	Key       string
	Item      *Item
	Watermark *int64

	finished bool
}

//go:tag serde:"json"
type Stats = map[string]int
