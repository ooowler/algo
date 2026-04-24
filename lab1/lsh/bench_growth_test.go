package lsh

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
)

func growthDatasetSizes() []int {
	return []int{1000, 2000, 4000, 8000, 12000, 16000, 24000, 32000}
}

func BenchmarkGrowthLSH_Build(b *testing.B) {
	const threshold = 1.5
	for _, n := range growthDatasetSizes() {
		points := GenerateDataset(n, threshold, int64(n))
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = BuildIndex(10, 8, threshold, rand.New(rand.NewSource(int64(1000+i))), points)
			}
		})
	}
}

func BenchmarkGrowthLSH_Add(b *testing.B) {
	const threshold = 1.5
	for _, n := range growthDatasetSizes() {
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			basePoints := GenerateDataset(n, threshold, int64(10+n))
			addPoints := GenerateDataset(b.N+64, threshold, int64(20+n))
			index := BuildIndex(10, 8, threshold, rand.New(rand.NewSource(int64(30+n))), basePoints)
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				index.Add(addPoints[i])
			}
		})
	}
}

func BenchmarkGrowthLSH_Find(b *testing.B) {
	const threshold = 1.5
	for _, n := range growthDatasetSizes() {
		points := GenerateDataset(n, threshold, int64(40+n))
		index := BuildIndex(10, 8, threshold, rand.New(rand.NewSource(int64(50+n))), points)
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				index.FindDuplicates()
			}
		})
	}
}

func BenchmarkGrowthLSH_Naive(b *testing.B) {
	const threshold = 1.5
	for _, n := range growthDatasetSizes() {
		points := GenerateDataset(n, threshold, int64(60+n))
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			runtime.GC()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				NaiveFindDuplicates(points, threshold)
			}
		})
	}
}
