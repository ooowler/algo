package perfecthash

import (
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkGrowthPH_Build(b *testing.B) {
	sizes := []int{256, 2048, 16384, 65536, 262144, 1048576}
	for _, n := range sizes {
		keys := make([]string, n)
		vals := make([]string, n)
		for i := range keys {
			keys[i] = strconv.Itoa(i)
			vals[i] = strconv.Itoa(i * 2)
		}
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Build(keys, vals)
			}
		})
	}
}

func BenchmarkGrowthPH_Get(b *testing.B) {
	sizes := []int{256, 2048, 16384, 65536, 262144, 1048576}
	for _, n := range sizes {
		keys := make([]string, n)
		vals := make([]string, n)
		for i := range keys {
			keys[i] = strconv.Itoa(i)
			vals[i] = strconv.Itoa(i)
		}
		idx := Build(keys, vals)
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx.Get(keys[i%n])
			}
		})
	}
}
