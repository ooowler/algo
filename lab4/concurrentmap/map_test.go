package concurrentmap

import (
	"math/rand"
	"testing"
)

func TestPutGetSizeAndClear(t *testing.T) {
	m := NewStringMap[int](4)
	m.Put("alpha", 1)
	m.Put("beta", 2)
	m.Put("alpha", 7)

	if got, ok := m.Get("alpha"); !ok || got != 7 {
		t.Fatalf("alpha: got %d ok=%v", got, ok)
	}
	if got, ok := m.Get("beta"); !ok || got != 2 {
		t.Fatalf("beta: got %d ok=%v", got, ok)
	}
	if _, ok := m.Get("missing"); ok {
		t.Fatal("missing key should not exist")
	}
	if got := m.Size(); got != 2 {
		t.Fatalf("size: got %d want 2", got)
	}

	m.Clear()
	if got := m.Size(); got != 0 {
		t.Fatalf("size after clear: got %d want 0", got)
	}
	if _, ok := m.Get("alpha"); ok {
		t.Fatal("clear should remove data")
	}
}

func TestMergeExistingAndMissing(t *testing.T) {
	m := NewStringMap[int](4)
	if got := m.Merge("hits", 3, func(a, b int) int { return a + b }); got != 3 {
		t.Fatalf("merge missing: got %d want 3", got)
	}
	if got := m.Merge("hits", 5, func(a, b int) int { return a + b }); got != 8 {
		t.Fatalf("merge existing: got %d want 8", got)
	}
	if got, ok := m.Get("hits"); !ok || got != 8 {
		t.Fatalf("get after merge: got %d ok=%v", got, ok)
	}
}

func TestIteratorIncludesAllPairs(t *testing.T) {
	m := NewStringMap[int](4)
	want := map[string]int{
		"alpha": 1,
		"beta":  2,
		"gamma": 3,
	}
	for key, value := range want {
		m.Put(key, value)
	}

	got := make(map[string]int, len(want))
	it := m.Iterator()
	for {
		item, ok := it.Next()
		if !ok {
			break
		}
		got[item.Key] = item.Value
	}
	if len(got) != len(want) {
		t.Fatalf("iterator len: got %d want %d", len(got), len(want))
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("iterator key %q: got %d want %d", key, got[key], value)
		}
	}
}

func TestGrowthKeepsValues(t *testing.T) {
	m := NewStringMap[int](2)
	for i := 0; i < 20000; i++ {
		m.Put(testKey(i), i)
	}
	if m.BucketCount() <= minBucketCount {
		t.Fatalf("expected resize, bucket count=%d", m.BucketCount())
	}
	for i := 0; i < 20000; i++ {
		if got, ok := m.Get(testKey(i)); !ok || got != i {
			t.Fatalf("key=%d got=%d ok=%v", i, got, ok)
		}
	}
}

func TestRandomOperationsMatchReference(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := NewStringMap[int](16)
	ref := make(map[string]int)

	for step := 0; step < 5000; step++ {
		key := testKey(rng.Intn(512))
		switch rng.Intn(3) {
		case 0:
			value := rng.Intn(1_000_000)
			m.Put(key, value)
			ref[key] = value
		case 1:
			value := 1 + rng.Intn(9)
			got := m.Merge(key, value, func(a, b int) int { return a + b })
			ref[key] += value
			if got != ref[key] {
				t.Fatalf("step=%d merge key=%q got=%d want=%d", step, key, got, ref[key])
			}
		default:
			got, ok := m.Get(key)
			want, wantOK := ref[key]
			if ok != wantOK || got != want {
				t.Fatalf("step=%d get key=%q got=%d ok=%v want=%d wantOK=%v", step, key, got, ok, want, wantOK)
			}
		}

		if step%200 == 0 {
			if got := m.Size(); got != len(ref) {
				t.Fatalf("step=%d size got=%d want=%d", step, got, len(ref))
			}
			for key, want := range ref {
				got, ok := m.Get(key)
				if !ok || got != want {
					t.Fatalf("step=%d key=%q got=%d ok=%v want=%d", step, key, got, ok, want)
				}
			}
		}
	}
}
