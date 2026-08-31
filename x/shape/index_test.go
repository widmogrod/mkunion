package shape

import (
	"sync"
	"testing"
)

// TestIndexConcurrentUse exercises the Index from many goroutines at once.
// Run with -race; before the Index was guarded by a mutex this crashed
// with "fatal error: concurrent map read and map write".
func TestIndexConcurrentUse(t *testing.T) {
	ResetIndex()
	defer ResetIndex()

	files := []string{
		"testasset/type_example.go",
		"testasset/type_other.go",
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		for _, filename := range files {
			wg.Add(1)
			go func(filename string) {
				defer wg.Done()
				info, err := InferFromFile(filename)
				if err != nil {
					t.Errorf("InferFromFile(%s): %s", filename, err)
					return
				}
				if len(info.RetrieveShapes()) == 0 {
					t.Errorf("InferFromFile(%s): no shapes", filename)
				}
			}(filename)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			LookupPkgShapeOnDisk("github.com/widmogrod/mkunion/x/shape/testasset")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			DefaultIndex.Stats()
		}()
	}
	wg.Wait()
}
