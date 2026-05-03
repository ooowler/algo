package concurrentmap

import (
	"fmt"
	"runtime"
	"testing"
)

var growthSizes = []int{1024, 4096, 16384, 65536, 262144, 1048576}

func makeProbeKeys(keys []string, maxProbes int) []string {
	if len(keys) == 0 {
		return nil
	}
	count := len(keys)
	if count > maxProbes {
		count = maxProbes
	}
	count = nextPow2(count)
	out := make([]string, count)
	for i := range out {
		out[i] = keys[int((uint64(i)*11400714819323198485)%uint64(len(keys)))]
	}
	return out
}

func BenchmarkGrowthConcurrentGet(b *testing.B) {
	for _, n := range growthSizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			keys := makeKeys(0, n)
			m := NewStringMap[int](bucketCountFor(n))
			for i := range keys {
				m.Put(keys[i], i)
			}
			probes := makeProbeKeys(keys, 4096)
			mask := len(probes) - 1
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = m.Get(probes[i&mask])
			}
		})
	}
}

func BenchmarkGrowthPlainGet(b *testing.B) {
	for _, n := range growthSizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			keys := makeKeys(0, n)
			m := newPlainStringMap[int](bucketCountFor(n))
			for i := range keys {
				m.Put(keys[i], i)
			}
			probes := makeProbeKeys(keys, 4096)
			mask := len(probes) - 1
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = m.Get(probes[i&mask])
			}
		})
	}
}

func BenchmarkGrowthConcurrentPut(b *testing.B) {
	for _, n := range growthSizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			keys := makeKeys(0, n)
			insert := makeKeys(n, b.N)
			m := NewStringMap[int](bucketCountFor(n + b.N))
			for i := range keys {
				m.Put(keys[i], i)
			}
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m.Put(insert[i], i)
			}
		})
	}
}

func BenchmarkGrowthPlainPut(b *testing.B) {
	for _, n := range growthSizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			keys := makeKeys(0, n)
			insert := makeKeys(n, b.N)
			m := newPlainStringMap[int](bucketCountFor(n + b.N))
			for i := range keys {
				m.Put(keys[i], i)
			}
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m.Put(insert[i], i)
			}
		})
	}
}

func BenchmarkGrowthConcurrentMerge(b *testing.B) {
	for _, n := range growthSizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			keys := makeKeys(0, n)
			m := NewStringMap[int](bucketCountFor(n))
			for i := range keys {
				m.Put(keys[i], 1)
			}
			probes := makeProbeKeys(keys, 4096)
			mask := len(probes) - 1
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m.Merge(probes[i&mask], 1, func(a, b int) int { return a + b })
			}
		})
	}
}

func BenchmarkGrowthPlainMerge(b *testing.B) {
	for _, n := range growthSizes {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			keys := makeKeys(0, n)
			m := newPlainStringMap[int](bucketCountFor(n))
			for i := range keys {
				m.Put(keys[i], 1)
			}
			probes := makeProbeKeys(keys, 4096)
			mask := len(probes) - 1
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m.Merge(probes[i&mask], 1, func(a, b int) int { return a + b })
			}
		})
	}
}

func BenchmarkParallelReadMostly(b *testing.B) {
	keys := makeKeys(0, 1<<16)
	m := NewStringMap[int](bucketCountFor(len(keys)))
	for i := range keys {
		m.Put(keys[i], i)
	}
	mask := len(keys) - 1
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := keys[i&mask]
			if i&15 == 0 {
				m.Merge(key, 1, func(a, b int) int { return a + b })
			} else {
				_, _ = m.Get(key)
			}
			i++
		}
	})
}

func BenchmarkParallelBalanced(b *testing.B) {
	keys := makeKeys(0, 1<<16)
	m := NewStringMap[int](bucketCountFor(len(keys)))
	for i := range keys {
		m.Put(keys[i], i)
	}
	mask := len(keys) - 1
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := keys[i&mask]
			if i&1 == 0 {
				m.Merge(key, 1, func(a, b int) int { return a + b })
			} else {
				_, _ = m.Get(key)
			}
			i++
		}
	})
}
