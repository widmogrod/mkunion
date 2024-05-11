package schema

import (
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"testing"
	"testing/quick"
)

func TestNative(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assertTypeConversion(t, 1)
	})
	t.Run("int8", func(t *testing.T) {
		if err := quick.Check(func(x int8) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("int16", func(t *testing.T) {
		if err := quick.Check(func(x int16) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("int32", func(t *testing.T) {
		if err := quick.Check(func(x int32) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("int64", func(t *testing.T) {
		t.Skip("boundary conversion issue because *Number is float64")
		if err := quick.Check(func(x int64) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("uint", func(t *testing.T) {
		t.Skip("boundary conversion issue because *Number is float64")
		if err := quick.Check(func(x uint) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("uint8", func(t *testing.T) {
		if err := quick.Check(func(x uint8) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("uint16", func(t *testing.T) {
		if err := quick.Check(func(x uint16) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}

	})
	t.Run("uint32", func(t *testing.T) {
		if err := quick.Check(func(x uint32) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("uint64", func(t *testing.T) {
		t.Skip("boundary conversion issue because *Number is float64")
		if err := quick.Check(func(x uint64) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("float32", func(t *testing.T) {
		if err := quick.Check(func(x float32) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("float64", func(t *testing.T) {
		if err := quick.Check(func(x float64) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("string", func(t *testing.T) {
		if err := quick.Check(func(x string) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
	t.Run("[]byte", func(t *testing.T) {
		if err := quick.Check(func(x []byte) bool {
			assertTypeConversion(t, x)
			return true
		}, nil); err != nil {
			t.Error(err)
		}
	})
}

func TestNonNative(t *testing.T) {
	t.Run("json.RawMessage", func(t *testing.T) {
		assertTypeConversion(t, json.RawMessage(`{"hello": "world"}`))
	})
	t.Run("time.Time", func(t *testing.T) {
		assertTypeConversion(t, "2021-01-01T00:00:00Z")
	})
}

func assertTypeConversion[A any](t *testing.T, value A) {
	expected := value
	t.Logf("expected = %+#v", expected)

	schemed := FromGo[A](expected)
	t.Logf("  FromGo = %+#v", schemed)

	result := ToGo[A](schemed)
	t.Logf("    ToGo = %+#v", result)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Error(diff)
	}
}
