package shape

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.ErrorLevel)
}

// BenchmarkIndexHotHit measures a cached file lookup (the common path).
func BenchmarkIndexHotHit(b *testing.B) {
	ResetIndex()
	if _, err := InferFromFile("testasset/type_example.go"); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InferFromFile("testasset/type_example.go")
	}
}

// BenchmarkIndexColdParse measures a full parse of a package (cache dropped each time).
func BenchmarkIndexColdParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetIndex()
		LookupPkgShapeOnDisk("github.com/widmogrod/mkunion/x/shape/testasset")
	}
}

// BenchmarkIndexPkgHit measures a cached package lookup.
func BenchmarkIndexPkgHit(b *testing.B) {
	ResetIndex()
	LookupPkgShapeOnDisk("github.com/widmogrod/mkunion/x/shape/testasset")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LookupPkgShapeOnDisk("github.com/widmogrod/mkunion/x/shape/testasset")
	}
}
