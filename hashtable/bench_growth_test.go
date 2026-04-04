package hashtable

import (
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkGrowthDisk_Set(b *testing.B) {
	sizes := []int{128, 1024, 8000, 32000, 65536}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			h, _ := New(b.TempDir(), 256)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h.Set(strconv.Itoa(i%n), "v")
			}
		})
	}
}

func BenchmarkGrowthDisk_Get(b *testing.B) {
	sizes := []int{128, 1024, 8000, 32000, 65536}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			h, _ := New(b.TempDir(), 256)
			for i := 0; i < n; i++ {
				h.Set(strconv.Itoa(i), "v")
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h.Get(strconv.Itoa(i % n))
			}
		})
	}
}

func BenchmarkGrowthDisk_Delete(b *testing.B) {
	sizes := []int{128, 1024, 8000, 16000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			h, _ := New(b.TempDir(), 256)
			for i := 0; i < b.N; i++ {
				h.Set(strconv.Itoa(i), "v")
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h.Delete(strconv.Itoa(i))
			}
		})
	}
}
