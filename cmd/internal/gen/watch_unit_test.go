package gen

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mapLen(m interface {
	Range(func(key, value any) bool)
}) int {
	n := 0
	m.Range(func(key, value any) bool {
		n++
		return true
	})
	return n
}

func TestWatchLoopHandleEvent(t *testing.T) {
	useCases := map[string]struct {
		event       fsnotify.Event
		wantChanged int
		wantRemoved int
	}{
		"non-go files are ignored": {
			fsnotify.Event{Name: "README.md", Op: fsnotify.Write}, 0, 0,
		},
		"chmod is ignored": {
			fsnotify.Event{Name: "a.go", Op: fsnotify.Chmod}, 0, 0,
		},
		"write marks the file changed": {
			fsnotify.Event{Name: "a.go", Op: fsnotify.Write}, 1, 0,
		},
		"create marks the file changed": {
			fsnotify.Event{Name: "a.go", Op: fsnotify.Create}, 1, 0,
		},
		"remove marks the file removed": {
			fsnotify.Event{Name: "a.go", Op: fsnotify.Remove}, 0, 1,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			w := &watchLoop{}
			w.handleEvent(uc.event)
			assert.Equal(t, uc.wantChanged, mapLen(&w.justChanged))
			assert.Equal(t, uc.wantRemoved, mapLen(&w.justRemoved))
		})
	}

	// Documents current behavior: the extension check runs before quotes
	// are trimmed, so a quoted name never matches ".go" and is dropped.
	t.Run("quoted event names are dropped by the extension check", func(t *testing.T) {
		w := &watchLoop{}
		w.handleEvent(fsnotify.Event{Name: `"a.go"`, Op: fsnotify.Write})
		assert.Equal(t, 0, mapLen(&w.justChanged))
	})

	t.Run("self-generated files get their mark cleared after a debounce", func(t *testing.T) {
		w := &watchLoop{}
		w.justGenerated.Store("a_union_gen.go", true)
		w.handleEvent(fsnotify.Event{Name: "a_union_gen.go", Op: fsnotify.Write})
		// the mark survives immediately (that is what prevents the loop)...
		_, ok := w.justGenerated.Load("a_union_gen.go")
		assert.True(t, ok)
		// ...and is dropped a debounce later so future edits regenerate
		assert.Eventually(t, func() bool {
			_, ok := w.justGenerated.Load("a_union_gen.go")
			return !ok
		}, 5*time.Second, 50*time.Millisecond)
	})
}

func TestWatchLoopProcessChanged(t *testing.T) {
	t.Run("nothing accumulated is a no-op", func(t *testing.T) {
		w := &watchLoop{}
		assert.NotPanics(t, func() { w.processChanged() })
	})

	t.Run("changed source regenerates and marks outputs as self-generated", func(t *testing.T) {
		sourcePath := writeVehicleModule(t)
		w := &watchLoop{}
		w.justChanged.Store(sourcePath, true)

		w.processChanged()

		unionPath := filepath.Join(filepath.Dir(sourcePath), "vehicle_union_gen.go")
		_, err := os.Stat(unionPath)
		require.NoError(t, err, "union file must be generated")

		_, selfGenerated := w.justGenerated.Load(unionPath)
		assert.True(t, selfGenerated, "generated files must not re-trigger generation")
		assert.Equal(t, 0, mapLen(&w.justChanged), "accumulated changes must be drained")
	})

	t.Run("self-generated changes do not regenerate", func(t *testing.T) {
		sourcePath := writeVehicleModule(t)
		w := &watchLoop{}
		w.justGenerated.Store(sourcePath, true)
		w.justChanged.Store(sourcePath, true)

		w.processChanged()

		unionPath := filepath.Join(filepath.Dir(sourcePath), "vehicle_union_gen.go")
		_, err := os.Stat(unionPath)
		assert.True(t, os.IsNotExist(err), "self-generated file must not trigger generation")
	})
}

func TestWatchLoopProcessRemoved(t *testing.T) {
	t.Run("nothing accumulated is a no-op", func(t *testing.T) {
		w := &watchLoop{}
		assert.NotPanics(t, func() { w.processRemoved() })
	})

	t.Run("removed file triggers a registry rebuild and drains the queue", func(t *testing.T) {
		sourcePath := writeVehicleModule(t)
		w := &watchLoop{}
		w.justRemoved.Store(filepath.Join(filepath.Dir(sourcePath), "gone.go"), true)

		assert.NotPanics(t, func() { w.processRemoved() })
		assert.Equal(t, 0, mapLen(&w.justRemoved), "accumulated removals must be drained")
	})

	t.Run("self-generated removals are ignored", func(t *testing.T) {
		sourcePath := writeVehicleModule(t)
		gone := filepath.Join(filepath.Dir(sourcePath), "gone.go")
		w := &watchLoop{}
		w.justGenerated.Store(gone, true)
		w.justRemoved.Store(gone, true)

		w.processRemoved()

		regPath := filepath.Join(filepath.Dir(sourcePath), "types_reg_gen.go")
		_, err := os.Stat(regPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestWatchLoopRunStops(t *testing.T) {
	t.Run("context cancellation stops the loop", func(t *testing.T) {
		watcher, err := fsnotify.NewWatcher()
		require.NoError(t, err)
		defer watcher.Close()

		w := &watchLoop{watcher: watcher}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan struct{})
		go func() {
			w.run(ctx)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("run ignored the cancelled context")
		}
	})

	t.Run("closed watcher stops the loop", func(t *testing.T) {
		watcher, err := fsnotify.NewWatcher()
		require.NoError(t, err)

		w := &watchLoop{watcher: watcher}
		done := make(chan struct{})
		go func() {
			w.run(context.Background())
			close(done)
		}()
		require.NoError(t, watcher.Close())
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("run did not stop when the watcher closed")
		}
	})
}
