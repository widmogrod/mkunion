package projection

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"sync"
	"testing"
	"time"
)

func Each(x schema.Schema, f func(value schema.Schema)) {
	_ = schema.MatchSchemaR1(
		x,
		func(x *schema.None) any {
			return nil
		},
		func(x *schema.Bool) any {
			f(x)
			return nil
		},
		func(x *schema.Number) any {
			f(x)
			return nil
		},
		func(x *schema.String) any {
			f(x)
			return nil
		},
		func(x *schema.Binary) any {
			f(x)
			return nil
		},
		func(x *schema.List) any {
			for _, v := range *x {
				f(v)
			}
			return nil
		},
		func(x *schema.Map) any {
			f(x)
			return nil
		},
	)
}

func GenerateItemsEvery(start int64, size int, every time.Duration) chan Item {
	ch := make(chan Item)
	t := time.Unix(0, start)

	go func() {
		defer close(ch)
		for i := 0; i < size; i++ {
			ch <- Item{
				//Key:       "key-" + strconv.Itoa(i),
				Key:       "key",
				Data:      schema.MkInt(int64(i)),
				EventTime: t.UnixNano(),
			}
			t = t.Add(every)
			//time.Sleep(every)
		}
	}()
	return ch
}

func NewDual() *Dual {
	return &Dual{}
}

type Dual struct {
	lock sync.Mutex
	list []*Message

	aggIdx int
	retIdx int
}

func (d *Dual) ReturningAggregate(msg Item) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.list = append(d.list, &Message{
		Key:  msg.Key,
		Item: &msg,
	})

	d.aggIdx++
}

func (d *Dual) ReturningRetract(msg Item) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.retIdx <= len(d.list) {
		if d.list[d.retIdx].Key != msg.Key {
			panic("key mismatch")
		}

		//d.list[d.retIdx].Watermark = &msg
		d.retIdx++
	}
}

func (d *Dual) IsValid() bool {
	return d.aggIdx == d.retIdx
}

func (d *Dual) List() []*Message {
	return d.list
}

type ListAssert struct {
	t     *testing.T
	Items []Item
	Err   error
}

func (l *ListAssert) Returning(msg Item) {
	if l.Err != nil {
		panic(l.Err)
	}

	l.Items = append(l.Items, msg)
}

func (l *ListAssert) AssertLen(expected int) bool {
	return assert.Equal(l.t, expected, len(l.Items))
}

func (l *ListAssert) AssertAt(index int, expected Item) bool {
	l.t.Helper()
	if diff := cmp.Diff(expected, l.Items[index]); diff != "" {
		l.t.Fatalf("mismatch at index %d (-want +got):\n%s", index, diff)
		return false
	}

	return true
}

func (l *ListAssert) Contains(expected Item) bool {
	l.t.Helper()
	for _, item := range l.Items {
		if diff := cmp.Diff(expected, item); diff == "" {
			return true
		}
	}

	l.t.Fatalf("projection to find %v in result set but failed", expected)
	return false
}
