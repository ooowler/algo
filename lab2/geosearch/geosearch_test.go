package geosearch

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

func randomPoints(n int, rng *rand.Rand) []Point {
	pts := make([]Point, n)
	for i := range pts {
		pts[i] = Point{ID: i, Lat: rng.Float64()*170 - 85, Lng: rng.Float64()*360 - 180}
	}
	return pts
}

func resultIDs(pts []Point) []int {
	ids := make([]int, len(pts))
	for i, p := range pts {
		ids[i] = p.ID
	}
	sort.Ints(ids)
	return ids
}

func assertMatchesNaive(t *testing.T, idx Index, pts []Point, center Point, radiusM float64) {
	t.Helper()
	naive := &NaiveIndex{}
	for _, p := range pts {
		idx.Add(p)
		naive.Add(p)
	}
	got := resultIDs(idx.Search(center, radiusM))
	want := resultIDs(naive.Search(center, radiusM))
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("id[%d]: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestDistanceM_ZeroForSamePoint(t *testing.T) {
	p := Point{Lat: 55.75, Lng: 37.62}
	if DistanceM(p, p) != 0 {
		t.Fatal("expected 0 for same point")
	}
}

func TestDistanceM_MoscowToSPb(t *testing.T) {
	moscow := Point{Lat: 55.75, Lng: 37.62}
	spb := Point{Lat: 59.95, Lng: 30.32}
	d := DistanceM(moscow, spb)
	if d < 630000 || d > 640000 {
		t.Fatalf("expected ~635km, got %.0fm", d)
	}
}

func TestKDTree_MatchesNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	pts := randomPoints(500, rng)
	assertMatchesNaive(t, &KDTree{}, pts, Point{Lat: 0, Lng: 0}, 500000)
}

func TestKDTree_LargeRadius_MatchesNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	pts := randomPoints(500, rng)
	assertMatchesNaive(t, &KDTree{}, pts, Point{Lat: 30, Lng: 30}, 5000000)
}

func TestKDTree_Build_MatchesNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	pts := randomPoints(1000, rng)
	naive := &NaiveIndex{}
	for _, p := range pts {
		naive.Add(p)
	}
	kd := Build(pts)
	center := Point{Lat: 10, Lng: 10}
	got := resultIDs(kd.Search(center, 1000000))
	want := resultIDs(naive.Search(center, 1000000))
	if len(got) != len(want) {
		t.Fatalf("Build len: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("Build id[%d]: got %d want %d", i, got[i], want[i])
		}
	}
}

func TestGrid_MatchesNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	pts := randomPoints(500, rng)
	assertMatchesNaive(t, NewGrid(0.5), pts, Point{Lat: 0, Lng: 0}, 500000)
}

func TestGrid_LargeRadius_MatchesNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	pts := randomPoints(500, rng)
	assertMatchesNaive(t, NewGrid(2.0), pts, Point{Lat: 30, Lng: 30}, 5000000)
}

func TestIndexes_HandleDateLine(t *testing.T) {
	pts := []Point{
		{ID: 1, Lat: 0, Lng: 179.95},
		{ID: 2, Lat: 0, Lng: -179.95},
		{ID: 3, Lat: 0, Lng: 179.0},
	}
	center := Point{Lat: 0, Lng: 179.99}
	radiusM := 20000.0
	for _, idx := range []Index{&KDTree{}, NewGrid(0.25)} {
		for _, p := range pts {
			idx.Add(p)
		}
		got := resultIDs(idx.Search(center, radiusM))
		want := []int{1, 2}
		if len(got) != len(want) {
			t.Fatalf("dateline len: got %v want %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("dateline ids: got %v want %v", got, want)
			}
		}
	}
}

func TestIndexes_RandomQueriesMatchNaive(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	pts := randomPoints(2000, rng)
	buildKD := Build(pts)
	grid := NewGrid(0.5)
	naive := &NaiveIndex{}
	for _, p := range pts {
		grid.Add(p)
		naive.Add(p)
	}
	for i := 0; i < 100; i++ {
		center := Point{
			Lat: rng.Float64()*170 - 85,
			Lng: math.Mod(rng.Float64()*400-200+180, 360) - 180,
		}
		radius := 1000 + rng.Float64()*4_000_000
		want := resultIDs(naive.Search(center, radius))
		for name, idx := range map[string]Index{
			"kd-build": buildKD,
			"grid":     grid,
		} {
			got := resultIDs(idx.Search(center, radius))
			if len(got) != len(want) {
				t.Fatalf("%s len: got %d want %d", name, len(got), len(want))
			}
			for j := range got {
				if got[j] != want[j] {
					t.Fatalf("%s id[%d]: got %d want %d", name, j, got[j], want[j])
				}
			}
		}
	}
}

func TestSearch_EmptyResult_WhenNoPointsInRadius(t *testing.T) {
	pt := Point{ID: 1, Lat: 80, Lng: 80}
	center := Point{Lat: 0, Lng: 0}
	for _, idx := range []Index{&KDTree{}, NewGrid(0.5)} {
		idx.Add(pt)
		if got := idx.Search(center, 100); len(got) != 0 {
			t.Fatalf("expected empty, got %d points", len(got))
		}
	}
}
