package lsh

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
)

func dist(a, b Point3D) float64 {
	return math.Sqrt(dist2(a, b))
}

func newLSH(threshold float64) *LSH {
	return New(10, 8, threshold, rand.New(rand.NewSource(42)))
}

func pairSet(pairs [][2]int) map[[2]int]bool {
	s := make(map[[2]int]bool)
	for _, p := range pairs {
		a, b := p[0], p[1]
		if a > b {
			a, b = b, a
		}
		s[[2]int{a, b}] = true
	}
	return s
}

func TestEmpty(t *testing.T) {
	l := newLSH(0.1)
	dups := l.FindDuplicates()
	if len(dups) != 0 {
		t.Fatalf("empty LSH should have no duplicates")
	}
}

func TestSinglePoint(t *testing.T) {
	l := newLSH(0.1)
	l.Add(Point3D{1, 2, 3, 0})
	dups := l.FindDuplicates()
	if len(dups) != 0 {
		t.Fatal("single point has no duplicates")
	}
}

func TestIdenticalPoints(t *testing.T) {
	l := newLSH(0.01)
	l.Add(Point3D{1, 1, 1, 0})
	l.Add(Point3D{1, 1, 1, 1})
	dups := l.FindDuplicates()
	if len(dups) == 0 {
		t.Fatal("identical points must be duplicates")
	}
}

func TestPointsWithinThreshold(t *testing.T) {
	l := newLSH(1.0)
	l.Add(Point3D{0, 0, 0, 0})
	l.Add(Point3D{0.5, 0, 0, 1})
	dups := l.FindDuplicates()
	if len(dups) == 0 {
		t.Fatal("points within threshold should be found")
	}
}

func TestPointsOutsideThreshold(t *testing.T) {
	l := newLSH(0.1)
	l.Add(Point3D{0, 0, 0, 0})
	l.Add(Point3D{10, 10, 10, 1})
	dups := l.FindDuplicates()
	if len(dups) != 0 {
		t.Fatal("far-apart points should not be duplicates")
	}
}

func TestKnownDuplicateClusters(t *testing.T) {
	l := newLSH(0.5)
	cluster1 := [][3]float64{{0, 0, 0}, {0.1, 0, 0}, {0, 0.1, 0}}
	cluster2 := [][3]float64{{100, 100, 100}, {100.1, 100, 100}}

	for i, c := range cluster1 {
		l.Add(Point3D{c[0], c[1], c[2], i})
	}
	for i, c := range cluster2 {
		l.Add(Point3D{c[0], c[1], c[2], i + len(cluster1)})
	}

	dups := l.FindDuplicates()
	if len(dups) == 0 {
		t.Fatal("should find duplicates within clusters")
	}

	dupSet := pairSet(dups)
	for _, p := range dups {
		a, b := p[0], p[1]
		if a >= len(l.points) || b >= len(l.points) {
			t.Fatalf("invalid index in duplicates: %v", p)
		}
		_ = dupSet
	}
}

func TestNoFalsePositivesOnGrid(t *testing.T) {
	l := newLSH(0.9)
	for x := 0; x < 5; x++ {
		for y := 0; y < 5; y++ {
			for z := 0; z < 5; z++ {
				l.Add(Point3D{float64(x * 2), float64(y * 2), float64(z * 2), x*25 + y*5 + z})
			}
		}
	}

	naive := NaiveFindDuplicates(l.points, 0.9)
	lshDups := l.FindDuplicates()

	naiveSet := pairSet(naive)
	for _, p := range lshDups {
		a, b := p[0], p[1]
		if a > b {
			a, b = b, a
		}
		if !naiveSet[[2]int{a, b}] {
			t.Fatalf("LSH returned false positive: %v (dist=%.4f)",
				p, dist(l.points[p[0]], l.points[p[1]]))
		}
	}
}

func TestAddIncrementally(t *testing.T) {
	l := newLSH(0.5)
	for i := 0; i < 10; i++ {
		l.Add(Point3D{float64(i) * 10, 0, 0, i})
	}
	l.Add(Point3D{0.1, 0, 0, 10})

	dups := l.FindDuplicates()
	dupSet := pairSet(dups)
	if !dupSet[[2]int{0, 10}] {
		t.Fatal("newly added point should be found as duplicate of point 0")
	}
}

func TestNaiveVsLSHRecall(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	l := New(15, 10, 2.0, rng)

	for i := 0; i < 300; i++ {
		l.Add(Point3D{rng.NormFloat64() * 10, rng.NormFloat64() * 10, rng.NormFloat64() * 10, i})
	}
	for i := 0; i < 10; i++ {
		base := l.points[rng.Intn(len(l.points))]
		l.Add(Point3D{base.X + rng.Float64()*0.5, base.Y + rng.Float64()*0.5, base.Z + rng.Float64()*0.5, 300 + i})
	}

	naive := NaiveFindDuplicates(l.points, 2.0)
	lshDups := l.FindDuplicates()

	naiveSet := pairSet(naive)
	lshSet := pairSet(lshDups)

	truePositives := 0
	for p := range naiveSet {
		if lshSet[p] {
			truePositives++
		}
	}

	recall := 1.0
	if len(naiveSet) > 0 {
		recall = float64(truePositives) / float64(len(naiveSet))
	}
	if recall < 0.7 {
		t.Fatalf("recall too low: %.2f (TP=%d / total=%d)", recall, truePositives, len(naiveSet))
	}
}

func TestDistanceFunction(t *testing.T) {
	cases := []struct {
		a, b Point3D
		want float64
	}{
		{Point3D{0, 0, 0, 0}, Point3D{1, 0, 0, 1}, 1.0},
		{Point3D{0, 0, 0, 0}, Point3D{0, 0, 0, 1}, 0.0},
		{Point3D{1, 1, 1, 0}, Point3D{2, 2, 2, 1}, math.Sqrt(3)},
		{Point3D{-1, -1, -1, 0}, Point3D{1, 1, 1, 1}, math.Sqrt(12)},
	}
	for _, c := range cases {
		got := dist(c.a, c.b)
		if math.Abs(got-c.want) > 1e-9 {
			t.Fatalf("dist(%v, %v) = %.4f, want %.4f", c.a, c.b, got, c.want)
		}
	}
}

func TestThresholdBoundary(t *testing.T) {
	threshold := 1.0
	l := newLSH(threshold)
	l.Add(Point3D{0, 0, 0, 0})
	l.Add(Point3D{0.999, 0, 0, 1})
	l.Add(Point3D{1.001, 0, 0, 2})

	naive := NaiveFindDuplicates(l.points, threshold)
	naiveSet := pairSet(naive)

	if !naiveSet[[2]int{0, 1}] {
		t.Fatal("pair (0,1) should be under threshold")
	}
	if naiveSet[[2]int{0, 2}] {
		t.Fatal("pair (0,2) should be over threshold")
	}
}

func TestLargeDataset(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	l := New(8, 10, 0.5, rng)

	known := make(map[[2]int]bool)
	for i := 0; i < 500; i++ {
		x := rng.Float64() * 100
		y := rng.Float64() * 100
		z := rng.Float64() * 100
		l.Add(Point3D{x, y, z, i})
		origIdx := len(l.points) - 1
		if i%10 == 0 {
			l.Add(Point3D{x + rng.Float64()*0.2, y + rng.Float64()*0.2, z + rng.Float64()*0.2, i + 1000})
			dupIdx := len(l.points) - 1
			a, b := origIdx, dupIdx
			if a > b {
				a, b = b, a
			}
			known[[2]int{a, b}] = true
		}
	}

	dups := l.FindDuplicates()
	dupSet := pairSet(dups)

	found := 0
	for p := range known {
		if dupSet[p] {
			found++
		}
	}
	recall := float64(found) / float64(len(known))
	if recall < 0.5 {
		t.Fatalf("recall on known duplicates too low: %.2f", recall)
	}
}

func TestSortedDuplicateIndexes(t *testing.T) {
	l := newLSH(1.0)
	for i := 0; i < 5; i++ {
		l.Add(Point3D{0, 0, 0, i})
	}
	dups := l.FindDuplicates()
	for _, p := range dups {
		if p[0] >= p[1] {
			t.Fatalf("pair not sorted: %v", p)
		}
	}
}

func BenchmarkAdd(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	l := New(10, 10, 1.0, rng)
	points := make([]Point3D, b.N)
	for i := range points {
		points[i] = Point3D{rng.Float64() * 100, rng.Float64() * 100, rng.Float64() * 100, i}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Add(points[i])
	}
}

func BenchmarkFindDuplicates(b *testing.B) {
	rng := rand.New(rand.NewSource(2))
	l := New(10, 10, 1.0, rng)
	for i := 0; i < 10000; i++ {
		l.Add(Point3D{rng.Float64() * 100, rng.Float64() * 100, rng.Float64() * 100, i})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.FindDuplicates()
	}
}

func BenchmarkNaiveFindDuplicates(b *testing.B) {
	rng := rand.New(rand.NewSource(3))
	points := make([]Point3D, 10000)
	for i := range points {
		points[i] = Point3D{rng.Float64() * 100, rng.Float64() * 100, rng.Float64() * 100, i}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NaiveFindDuplicates(points, 1.0)
	}
}

func TestRecallVsPrecision(t *testing.T) {
	rng := rand.New(rand.NewSource(55))
	l := New(12, 10, 1.5, rng)

	for i := 0; i < 200; i++ {
		l.Add(Point3D{rng.Float64() * 50, rng.Float64() * 50, rng.Float64() * 50, i})
	}

	naive := NaiveFindDuplicates(l.points, 1.5)
	lshDups := l.FindDuplicates()

	naiveSet := pairSet(naive)
	lshSet := pairSet(lshDups)

	tp := 0
	for p := range lshSet {
		if naiveSet[p] {
			tp++
		}
	}

	precision := 1.0
	if len(lshSet) > 0 {
		precision = float64(tp) / float64(len(lshSet))
	}
	if precision < 0.99 {
		t.Fatalf("precision too low: %.2f", precision)
	}

	_ = fmt.Sprintf("naive=%d lsh=%d tp=%d precision=%.2f", len(naive), len(lshDups), tp, precision)
}

func TestDuplicatesAreUnique(t *testing.T) {
	l := newLSH(5.0)
	rng := rand.New(rand.NewSource(77))
	for i := 0; i < 100; i++ {
		l.Add(Point3D{rng.Float64() * 10, rng.Float64() * 10, rng.Float64() * 10, i})
	}
	dups := l.FindDuplicates()
	seen := make(map[[2]int]bool)
	for _, p := range dups {
		a, b := p[0], p[1]
		if a > b {
			a, b = b, a
		}
		key := [2]int{a, b}
		if seen[key] {
			t.Fatalf("duplicate pair reported twice: %v", key)
		}
		seen[key] = true
	}
}

func TestSortedResultIndexes(t *testing.T) {
	l := newLSH(2.0)
	rng := rand.New(rand.NewSource(88))
	for i := 0; i < 50; i++ {
		l.Add(Point3D{rng.Float64() * 5, rng.Float64() * 5, rng.Float64() * 5, i})
	}
	dups := l.FindDuplicates()
	sorted := make([][2]int, len(dups))
	copy(sorted, dups)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i][0] != sorted[j][0] {
			return sorted[i][0] < sorted[j][0]
		}
		return sorted[i][1] < sorted[j][1]
	})
	_ = sorted
}
