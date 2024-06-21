package stats

import (
	"testing"
	"unsafe"
)

func TestFieldSize(t *testing.T) {
	size := unsafe.Sizeof(Field{})
	t.Log("field size:", size)
}

func BenchmarkAssign40BytesStruct(b *testing.B) {
	type S struct {
		a string
		b string
		c int
	}

	var s S

	for i := 0; i != b.N; i++ {
		s = S{a: "hello", b: "", c: 0}
		_ = s
	}
}

func BenchmarkAssign32BytesStruct(b *testing.B) {
	type S struct {
		a string
		b string
	}

	var s S

	for i := 0; i != b.N; i++ {
		s = S{a: "hello", b: ""}
		_ = s
	}
}
