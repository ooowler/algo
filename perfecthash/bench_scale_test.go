package perfecthash

import (
	"fmt"
	"strconv"
	"testing"
)

func BenchmarkBuild_1K(b *testing.B) {
	benchBuild(b, 1000)
}

func BenchmarkBuild_10K(b *testing.B) {
	benchBuild(b, 10000)
}

func BenchmarkBuild_50K(b *testing.B) {
	benchBuild(b, 50000)
}

func benchBuild(b *testing.B, n int) {
	keys := make([]string, n)
	vals := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("k-%d", i)
		vals[i] = strconv.Itoa(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Build(keys, vals)
	}
}

func BenchmarkGet_1K(b *testing.B) {
	benchGetN(b, 1000)
}

func BenchmarkGet_10K(b *testing.B) {
	benchGetN(b, 10000)
}

func BenchmarkGet_50K(b *testing.B) {
	benchGetN(b, 50000)
}

func benchGetN(b *testing.B, n int) {
	keys := make([]string, n)
	vals := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("k-%d", i)
		vals[i] = strconv.Itoa(i)
	}
	idx := Build(keys, vals)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Get(keys[i%n])
	}
}
