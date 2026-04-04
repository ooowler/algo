package lsh

import (
	"math"
	"math/rand"
)

const (
	KPROJ  = 3
	BANDK  = 3.0
)

type Point3D struct {
	X, Y, Z float64
	ID      int
}

type cell [KPROJ]int32

type grid struct {
	dirs    [KPROJ][3]float64
	offsets [KPROJ]float64
	width   float64
	buckets map[cell][]int
}

type LSH struct {
	tables      []*grid
	points      []Point3D
	thresholdSq float64
}

func New(numTables int, _ int, threshold float64, rng *rand.Rand) *LSH {
	w := threshold * BANDK
	tables := make([]*grid, numTables)
	for i := range tables {
		g := &grid{width: w, buckets: make(map[cell][]int)}
		for j := 0; j < KPROJ; j++ {
			x, y, z := rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()
			n := math.Sqrt(x*x + y*y + z*z)
			if n == 0 {
				n = 1
			}
			g.dirs[j] = [3]float64{x / n, y / n, z / n}
			g.offsets[j] = rng.Float64() * w
		}
		tables[i] = g
	}
	return &LSH{tables, nil, threshold * threshold}
}

func (g *grid) key(p Point3D) cell {
	var c cell
	inv := 1.0 / g.width
	for i := 0; i < KPROJ; i++ {
		d := p.X*g.dirs[i][0] + p.Y*g.dirs[i][1] + p.Z*g.dirs[i][2]
		c[i] = int32(math.Floor((d + g.offsets[i]) * inv))
	}
	return c
}

func (l *LSH) Add(p Point3D) {
	i := len(l.points)
	l.points = append(l.points, p)
	for _, g := range l.tables {
		k := g.key(p)
		g.buckets[k] = append(g.buckets[k], i)
	}
}

func dist2(a, b Point3D) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return dx*dx + dy*dy + dz*dz
}

func pairKey(lo, hi int) uint64 {
	return uint64(uint32(lo))<<32 | uint64(uint32(hi))
}

func (l *LSH) FindDuplicates() [][2]int {
	seen := make(map[uint64]struct{})
	var out [][2]int
	pts := l.points
	th := l.thresholdSq
	for _, g := range l.tables {
		for _, b := range g.buckets {
			for i := 0; i < len(b); i++ {
				for j := i + 1; j < len(b); j++ {
					a, c := b[i], b[j]
					if a > c {
						a, c = c, a
					}
					pk := pairKey(a, c)
					if _, dup := seen[pk]; dup {
						continue
					}
					seen[pk] = struct{}{}
					if dist2(pts[a], pts[c]) < th {
						out = append(out, [2]int{a, c})
					}
				}
			}
		}
	}
	return out
}

func NaiveFindDuplicates(points []Point3D, threshold float64) [][2]int {
	th := threshold * threshold
	var out [][2]int
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if dist2(points[i], points[j]) < th {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}
