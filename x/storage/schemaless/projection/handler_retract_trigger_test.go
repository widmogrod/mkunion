package projection

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

func intMerge() *MergeHandler[int] {
	return &MergeHandler[int]{
		Combine:   func(a, b int) (int, error) { return a + b, nil },
		DoRetract: func(a, b int) (int, error) { return a - b, nil },
	}
}

func TestMergeHandlerRetract(t *testing.T) {
	t.Run("retracts fold with DoRetract", func(t *testing.T) {
		l := &ListAssert{t: t}
		err := intMerge().Retract(Item{
			Key:  "k",
			Data: schema.MkList(schema.MkInt(5), schema.MkInt(2), schema.MkInt(1)),
		}, l.Returning)
		require.NoError(t, err)
		l.AssertAt(0, Item{
			Key:  "k",
			Data: schema.FromGo(2),
		})
	})

	t.Run("single element passes through", func(t *testing.T) {
		l := &ListAssert{t: t}
		err := intMerge().Retract(Item{
			Key:  "k",
			Data: schema.MkList(schema.MkInt(5)),
		}, l.Returning)
		require.NoError(t, err)
		l.AssertAt(0, Item{Key: "k", Data: schema.FromGo(5)})
	})

	t.Run("DoRetract error propagates", func(t *testing.T) {
		h := intMerge()
		h.DoRetract = func(a, b int) (int, error) { return 0, errors.New("cannot retract") }
		err := h.Retract(Item{
			Key:  "k",
			Data: schema.MkList(schema.MkInt(5), schema.MkInt(2)),
		}, func(Item) { t.Fatal("must not emit on error") })
		assert.ErrorContains(t, err, "cannot retract")
	})

	t.Run("unconvertible element errors", func(t *testing.T) {
		err := intMerge().Retract(Item{
			Key:  "k",
			Data: schema.MkList(schema.MkString("not-an-int"), schema.MkInt(2)),
		}, func(Item) { t.Fatal("must not emit on error") })
		assert.Error(t, err)
	})
}

func TestMergeHandlerProcessErrors(t *testing.T) {
	t.Run("Combine error propagates", func(t *testing.T) {
		h := intMerge()
		h.Combine = func(a, b int) (int, error) { return 0, errors.New("cannot combine") }
		err := h.Process(Item{
			Key:  "k",
			Data: schema.MkList(schema.MkInt(1), schema.MkInt(2)),
		}, func(Item) { t.Fatal("must not emit on error") })
		assert.ErrorContains(t, err, "cannot combine")
	})
}

func newTestTriggerHandler(td TriggerDescription) *TriggerHandler {
	tm := NewTriggerManager(td)
	tm.WhenTrigger(func(kw *KeyedWindow) {})
	return &TriggerHandler{
		td:  td,
		wb:  NewWindowBuffer(&FixedWindow{Width: time.Second}, tm),
		wts: NewInMemoryBagOf[*WindowTrigger](),
	}
}

func itemAt(key string, when int64, value int64) Item {
	return Item{
		Key:       key,
		Data:      schema.MkInt(value),
		EventTime: when,
	}
}

func TestTriggerHandlerTriggered(t *testing.T) {
	base := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()

	t.Run("window emits once the item-count trigger is reached", func(t *testing.T) {
		h := newTestTriggerHandler(&AtWindowItemSize{Number: 2})

		var emitted []Item
		returning := func(x Item) { emitted = append(emitted, x) }

		require.NoError(t, h.Process(itemAt("k", base, 1), returning))
		assert.Empty(t, emitted, "one item is below the trigger threshold")

		require.NoError(t, h.Process(itemAt("k", base+1, 2), returning))
		require.Len(t, emitted, 1, "second item reaches the threshold")
		assert.Equal(t, "k", emitted[0].Key)
		assert.Equal(t, schema.MkList(schema.MkInt(1), schema.MkInt(2)), emitted[0].Data)
	})

	t.Run("emitted windows are removed from the buffer", func(t *testing.T) {
		h := newTestTriggerHandler(&AtWindowItemSize{Number: 1})

		var emitted []Item
		returning := func(x Item) { emitted = append(emitted, x) }

		require.NoError(t, h.Process(itemAt("k", base, 1), returning))
		require.Len(t, emitted, 1)

		// triggering again must not re-emit the flushed window
		require.NoError(t, h.Triggered(&AtWatermark{}, returning))
		assert.Len(t, emitted, 1)
	})

	t.Run("separate keys emit separate windows", func(t *testing.T) {
		h := newTestTriggerHandler(&AtWindowItemSize{Number: 1})

		var emitted []Item
		returning := func(x Item) { emitted = append(emitted, x) }

		require.NoError(t, h.Process(itemAt("a", base, 1), returning))
		require.NoError(t, h.Process(itemAt("b", base, 2), returning))
		require.Len(t, emitted, 2)
		keys := []string{emitted[0].Key, emitted[1].Key}
		assert.ElementsMatch(t, []string{"a", "b"}, keys)
	})

	t.Run("watermark trigger without buffered items is a no-op", func(t *testing.T) {
		h := newTestTriggerHandler(&AtWatermark{})
		err := h.Triggered(&AtWatermark{}, func(Item) { t.Fatal("nothing to emit") })
		assert.NoError(t, err)
	})

	t.Run("retract is not implemented", func(t *testing.T) {
		h := newTestTriggerHandler(&AtWatermark{})
		assert.Panics(t, func() {
			_ = h.Retract(Item{}, func(Item) {})
		})
	})
}
