package geosearch

import (
	"fmt"
	"math/rand"
	"testing"
)

var buildSizes = []int{1000, 5000, 20000, 80000, 300000}
var searchSizes = []int{10000, 100000, 500000, 1000000}

const searchRadius = 500000.0

var benchCenter = Point{Lat: 48.85, Lng: 2.35}

const radiusN = 100000

var testRadii = []float64{10000, 100000, 500000, 2000000}

func genPoints(n int) []Point {
	rng := rand.New(rand.NewSource(42))
	pts := make([]Point, n)
	for i := range pts {
		pts[i] = Point{ID: i, Lat: rng.Float64()*170 - 85, Lng: rng.Float64()*360 - 180}
	}
	return pts
}

func BenchmarkGrowthKD_Build(b *testing.B) {
	for _, n := range buildSizes {
		pts := genPoints(n)
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Build(pts)
			}
		})
	}
}

func BenchmarkGrowthKD_Search(b *testing.B) {
	for _, n := range searchSizes {
		pts := genPoints(n)
		kd := Build(pts)
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				kd.Search(benchCenter, searchRadius)
			}
		})
	}
}

func BenchmarkGrowthGrid_Search(b *testing.B) {
	for _, n := range searchSizes {
		pts := genPoints(n)
		g := NewGrid(searchRadius / 111320.0)
		for _, p := range pts {
			g.Add(p)
		}
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				g.Search(benchCenter, searchRadius)
			}
		})
	}
}

func BenchmarkGrowthNaive_Search(b *testing.B) {
	for _, n := range searchSizes {
		pts := genPoints(n)
		naive := &NaiveIndex{}
		for _, p := range pts {
			naive.Add(p)
		}
		b.Run(fmt.Sprintf("N%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				naive.Search(benchCenter, searchRadius)
			}
		})
	}
}

func BenchmarkRadiusKD_Search(b *testing.B) {
	pts := genPoints(radiusN)
	kd := Build(pts)
	for _, r := range testRadii {
		r := r
		b.Run(fmt.Sprintf("R%d", int(r)), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				kd.Search(benchCenter, r)
			}
		})
	}
}

func BenchmarkRadiusGrid_Search(b *testing.B) {
	pts := genPoints(radiusN)
	for _, r := range testRadii {
		g := NewGrid(r / 111320.0)
		for _, p := range pts {
			g.Add(p)
		}
		r := r
		b.Run(fmt.Sprintf("R%d", int(r)), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				g.Search(benchCenter, r)
			}
		})
	}
}

func BenchmarkRadiusNaive_Search(b *testing.B) {
	pts := genPoints(radiusN)
	naive := &NaiveIndex{}
	for _, p := range pts {
		naive.Add(p)
	}
	for _, r := range testRadii {
		r := r
		b.Run(fmt.Sprintf("R%d", int(r)), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				naive.Search(benchCenter, r)
			}
		})
	}
}
