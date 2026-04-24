package perfecthash

import (
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
	keys, vals := makePHInput(n)
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
	keys, vals := makePHInput(n)
	idx := Build(keys, vals)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Get(keys[i%n])
	}
}
