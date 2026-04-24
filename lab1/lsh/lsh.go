package lsh

import (
	"math"
	"math/rand"
)

const (
	KPROJ  = 3
	BANDK  = 3.0

	maxDenseSeenBytes = 96 << 20
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
	invWidth float64
	buckets map[cell][]int
}

type LSH struct {
	tables      []*grid
	points      []Point3D
	thresholdSq float64
	seenScratch map[uint64]struct{}
	seenCap     int
	denseSeen   []uint64
}

func New(numTables int, expectedPoints int, threshold float64, rng *rand.Rand) *LSH {
	w := threshold * BANDK
	if expectedPoints < 0 {
		expectedPoints = 0
	}
	bucketHint := expectedPoints / 2
	if bucketHint < 16 {
		bucketHint = 16
	}
	tables := make([]*grid, numTables)
	for i := range tables {
		g := &grid{
			width:    w,
			invWidth: 1.0 / w,
			buckets:  make(map[cell][]int, bucketHint),
		}
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
	return &LSH{
		tables:      tables,
		points:      make([]Point3D, 0, expectedPoints),
		thresholdSq: threshold * threshold,
	}
}

func (g *grid) key(p Point3D) cell {
	var c cell
	for i := 0; i < KPROJ; i++ {
		d := p.X*g.dirs[i][0] + p.Y*g.dirs[i][1] + p.Z*g.dirs[i][2]
		c[i] = int32(math.Floor((d + g.offsets[i]) * g.invWidth))
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
	out, _ := l.FindDuplicatesWithStats()
	return out
}

type Stats struct {
	UniqueCandidates int
	TotalCandidates  int
}

func (l *LSH) FindDuplicatesWithStats() ([][2]int, Stats) {
	pts := l.points
	n := len(pts)
	outCap := n
	if outCap > 1<<18 {
		outCap = 1 << 18
	}
	out := make([][2]int, 0, outCap)
	th := l.thresholdSq
	total := 0
	unique := 0
	seenBits := l.acquireDenseSeen(n)
	var seen map[uint64]struct{}
	if seenBits == nil {
		seenCap := n * 4
		if seenCap < 64 {
			seenCap = 64
		}
		if seenCap > 1<<22 {
			seenCap = 1 << 22
		}
		seen = l.acquireSeenScratch(seenCap)
	}
	for _, g := range l.tables {
		for _, b := range g.buckets {
			for i := 0; i < len(b); i++ {
				a := b[i]
				pa := pts[a]
				for j := i + 1; j < len(b); j++ {
					c := b[j]
					lo, hi := a, c
					if lo > hi {
						lo, hi = hi, lo
					}
					total++
					if seenBits != nil {
						offset := pairOffset(n, lo, hi)
						word := offset >> 6
						mask := uint64(1) << (offset & 63)
						if seenBits[word]&mask != 0 {
							continue
						}
						seenBits[word] |= mask
					} else {
						pk := pairKey(lo, hi)
						if _, dup := seen[pk]; dup {
							continue
						}
						seen[pk] = struct{}{}
					}
					unique++
					pb := pts[c]
					dx, dy, dz := pa.X-pb.X, pa.Y-pb.Y, pa.Z-pb.Z
					if dx*dx+dy*dy+dz*dz < th {
						out = append(out, [2]int{lo, hi})
					}
				}
			}
		}
	}
	return out, Stats{UniqueCandidates: unique, TotalCandidates: total}
}

func (l *LSH) acquireSeenScratch(seenCap int) map[uint64]struct{} {
	if l.seenScratch == nil || l.seenCap < seenCap {
		l.seenScratch = make(map[uint64]struct{}, seenCap)
		l.seenCap = seenCap
		return l.seenScratch
	}
	clear(l.seenScratch)
	return l.seenScratch
}

func (l *LSH) acquireDenseSeen(n int) []uint64 {
	if n < 2 {
		return nil
	}
	pairs := int64(n) * int64(n-1) / 2
	words := int((pairs + 63) >> 6)
	if words <= 0 || words*8 > maxDenseSeenBytes {
		return nil
	}
	if cap(l.denseSeen) < words {
		l.denseSeen = make([]uint64, words)
	}
	seen := l.denseSeen[:words]
	clear(seen)
	return seen
}

func pairOffset(n, lo, hi int) int64 {
	return int64(lo)*(int64(2*n-lo-1))/2 + int64(hi-lo-1)
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
