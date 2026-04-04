package lsh

import (
	"fmt"
	"math/rand"
	"testing"
)

func BenchmarkGrowthLSH_Find(b *testing.B) {
	sizes := []int{400, 1600, 6400, 14000, 32000}
	rng := rand.New(rand.NewSource(7))
	for _, n := range sizes {
		l := New(10, 8, 1.5, rand.New(rand.NewSource(11)))
		for i := 0; i < n; i++ {
			l.Add(Point3D{
				X:  rng.Float64() * 80,
				Y:  rng.Float64() * 80,
				Z:  rng.Float64() * 80,
				ID: i,
			})
		}
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				l.FindDuplicates()
			}
		})
	}
}

func BenchmarkGrowthLSH_Naive(b *testing.B) {
	sizes := []int{400, 1600, 6400, 14000, 32000}
	rng := rand.New(rand.NewSource(13))
	for _, n := range sizes {
		points := make([]Point3D, n)
		for i := range points {
			points[i] = Point3D{
				X:  rng.Float64() * 80,
				Y:  rng.Float64() * 80,
				Z:  rng.Float64() * 80,
				ID: i,
			}
		}
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				NaiveFindDuplicates(points, 1.5)
			}
		})
	}
}
