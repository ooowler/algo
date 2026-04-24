package lsh

import (
	"math/rand"
	"testing"
)

func BenchmarkAdd_Small(b *testing.B) {
	benchAdd(b, 200)
}

func BenchmarkAdd_Medium(b *testing.B) {
	benchAdd(b, 5000)
}

func BenchmarkAdd_Large(b *testing.B) {
	benchAdd(b, 25000)
}

func benchAdd(b *testing.B, prefill int) {
	rng := rand.New(rand.NewSource(1))
	l := New(10, 8, 1.0, rng)
	for i := 0; i < prefill; i++ {
		l.Add(Point3D{rng.Float64() * 100, rng.Float64() * 100, rng.Float64() * 100, i})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Add(Point3D{rng.Float64() * 100, rng.Float64() * 100, rng.Float64() * 100, prefill + i})
	}
}

func BenchmarkFindDuplicates_Small(b *testing.B) {
	benchFindDup(b, 400)
}

func BenchmarkFindDuplicates_Medium(b *testing.B) {
	benchFindDup(b, 4000)
}

func BenchmarkFindDuplicates_Large(b *testing.B) {
	benchFindDup(b, 15000)
}

func benchFindDup(b *testing.B, n int) {
	rng := rand.New(rand.NewSource(42))
	l := New(10, 8, 1.0, rng)
	for i := 0; i < n; i++ {
		l.Add(Point3D{rng.Float64() * 100, rng.Float64() * 100, rng.Float64() * 100, i})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.FindDuplicates()
	}
}

func BenchmarkNaiveFindDuplicates_Small(b *testing.B) {
	benchNaiveDup(b, 400)
}

func BenchmarkNaiveFindDuplicates_Medium(b *testing.B) {
	benchNaiveDup(b, 2000)
}

func BenchmarkNaiveFindDuplicates_Large(b *testing.B) {
	benchNaiveDup(b, 6000)
}

func benchNaiveDup(b *testing.B, n int) {
	rng := rand.New(rand.NewSource(3))
	points := make([]Point3D, n)
	for i := range points {
		points[i] = Point3D{rng.Float64() * 100, rng.Float64() * 100, rng.Float64() * 100, i}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NaiveFindDuplicates(points, 1.0)
	}
}
