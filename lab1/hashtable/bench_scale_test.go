package hashtable

import (
	"testing"
)

func BenchmarkSet_Small(b *testing.B) {
	benchSet(b, 64, 100)
}

func BenchmarkSet_Medium(b *testing.B) {
	benchSet(b, 256, 2000)
}

func BenchmarkSet_Large(b *testing.B) {
	benchSet(b, 512, 50000)
}

func benchSet(b *testing.B, buckets, cycle int) {
	h, _ := New(b.TempDir(), buckets)
	defer h.Close()
	keys := makeBenchKeys(0, cycle)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Set(keys[i%cycle], "v")
	}
}

func BenchmarkGet_Small(b *testing.B) {
	benchGet(b, 64, 200)
}

func BenchmarkGet_Medium(b *testing.B) {
	benchGet(b, 256, 4000)
}

func BenchmarkGet_Large(b *testing.B) {
	benchGet(b, 512, 20000)
}

func benchGet(b *testing.B, buckets, n int) {
	h, _ := New(b.TempDir(), buckets)
	defer h.Close()
	keys := makeBenchKeys(0, n)
	for i := 0; i < n; i++ {
		h.Set(keys[i], "v")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Get(keys[i%n])
	}
}

func BenchmarkDelete_Small(b *testing.B) {
	benchDelete(b, 64)
}

func BenchmarkDelete_Medium(b *testing.B) {
	benchDelete(b, 256)
}

func BenchmarkDelete_Large(b *testing.B) {
	benchDelete(b, 512)
}

func benchDelete(b *testing.B, buckets int) {
	h, _ := New(b.TempDir(), buckets)
	defer h.Close()
	keys := makeBenchKeys(0, b.N)
	for i := 0; i < b.N; i++ {
		h.Set(keys[i], "v")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Delete(keys[i])
	}
}
