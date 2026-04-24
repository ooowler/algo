package hashtable

import (
	"fmt"
	"runtime"
	"testing"
)

func numBucketsFor(n int) int {
	buckets := n / 16
	if buckets < 64 {
		buckets = 64
	}
	return nextPow2(buckets)
}

func growthSizes() []int {
	return []int{2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144}
}

func benchKey(i int) string {
	return fmt.Sprintf("k%07d", i)
}

func benchValue(i int) string {
	return fmt.Sprintf("v%07d", i)
}

func makeBenchKeys(start, count int) []string {
	keys := make([]string, count)
	for i := 0; i < count; i++ {
		keys[i] = benchKey(start + i)
	}
	return keys
}

func makeProbeKeys(keys []string, maxProbes int) []string {
	if len(keys) == 0 {
		return nil
	}
	count := len(keys)
	if count > maxProbes {
		count = maxProbes
	}
	count = nextPow2(count)
	probes := make([]string, count)
	for i := range probes {
		idx := int((uint64(i) * 11400714819323198485) % uint64(len(keys)))
		probes[i] = keys[idx]
	}
	return probes
}

func nextPow2(v int) int {
	if v <= 1 {
		return 1
	}
	n := 1
	for n < v {
		n <<= 1
	}
	return n
}

func seedTable(h *DiskHashTable, keys []string) {
	perBucketCap := len(keys)/len(h.buckets) + 1
	for i := range h.buckets {
		h.buckets[i].loaded = true
		h.buckets[i].data = make(map[string]string, perBucketCap)
		h.meta[i].liveRecords = 0
		h.meta[i].staleRecords = 0
	}
	for _, key := range keys {
		idx := h.bucketIndex(key)
		bucket := &h.buckets[idx]
		meta := &h.meta[idx]
		bucket.data[key] = "value"
		meta.liveRecords++
	}
}

func BenchmarkGrowthDisk_Insert(b *testing.B) {
	for _, n := range growthSizes() {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			h, err := New(b.TempDir(), numBucketsFor(n+b.N))
			if err != nil {
				b.Fatal(err)
			}
			defer h.Close()
			seedTable(h, makeBenchKeys(0, n))
			insertKeys := makeBenchKeys(n, b.N)
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := h.Set(insertKeys[i], "value"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGrowthDisk_Update(b *testing.B) {
	for _, n := range growthSizes() {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			h, err := New(b.TempDir(), numBucketsFor(n))
			if err != nil {
				b.Fatal(err)
			}
			defer h.Close()
			keys := makeBenchKeys(0, n)
			seedTable(h, keys)
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := h.Set(keys[i%n], "updated"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGrowthDisk_Delete(b *testing.B) {
	for _, n := range growthSizes() {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			total := n + b.N
			h, err := New(b.TempDir(), numBucketsFor(total))
			if err != nil {
				b.Fatal(err)
			}
			defer h.Close()
			keys := makeBenchKeys(0, total)
			seedTable(h, keys)
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := h.Delete(keys[i]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGrowthDisk_Get(b *testing.B) {
	for _, n := range growthSizes() {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			h, err := New(b.TempDir(), numBucketsFor(n))
			if err != nil {
				b.Fatal(err)
			}
			defer h.Close()
			keys := makeBenchKeys(0, n)
			seedTable(h, keys)
			probes := makeProbeKeys(keys, 4096)
			probeMask := len(probes) - 1
			warm := len(probes) * 16
			if warm > 65536 {
				warm = 65536
			}
			for i := 0; i < warm; i++ {
				if _, _, err := h.Get(probes[i&probeMask]); err != nil {
					b.Fatal(err)
				}
			}
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, err := h.Get(probes[i&probeMask]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
