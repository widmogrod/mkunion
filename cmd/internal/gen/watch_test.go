package gen

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatch_RegeneratesOnFileChange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem watch test in short mode")
	}

	sourcePath := writeVehicleModule(t)
	dir := filepath.Dir(sourcePath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- Watch(ctx, []string{dir}, WatchOptions{
			TypeRegistry:  false,
			RunGoGenerate: false,
		})
	}()

	// give the watcher a moment to register the directory
	time.Sleep(500 * time.Millisecond)

	// touching the source file must trigger regeneration
	require.NoError(t, os.WriteFile(sourcePath, []byte(vehicleSource), 0644))

	unionPath := filepath.Join(dir, "vehicle_union_gen.go")
	require.Eventually(t, func() bool {
		_, err := os.Stat(unionPath)
		return err == nil
	}, 10*time.Second, 100*time.Millisecond, "expected %s to be generated", unionPath)

	cancel()
	select {
	case err := <-watchDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Watch did not stop after context cancellation")
	}
}
