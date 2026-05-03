package concurrentmap

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCompletedPutIsVisible(t *testing.T) {
	m := NewStringMap[int](4)
	done := make(chan struct{})
	go func() {
		m.Put("ready", 1)
		close(done)
	}()
	<-done
	if got, ok := m.Get("ready"); !ok || got != 1 {
		t.Fatalf("got %d ok=%v", got, ok)
	}
}

func TestConcurrentMergeMatchesReference(t *testing.T) {
	const (
		workers = 12
		ops     = 4000
		keys    = 64
	)

	m := NewStringMap[int](8)
	start := make(chan struct{})
	local := make([][]int, workers)
	var wg sync.WaitGroup
	var missing atomic.Bool

	for worker := 0; worker < workers; worker++ {
		local[worker] = make([]int, keys)
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(100 + id)))
			<-start
			for i := 0; i < ops; i++ {
				keyID := rng.Intn(keys)
				key := testKey(keyID)
				local[id][keyID]++
				m.Merge(key, 1, func(a, b int) int { return a + b })
				if _, ok := m.Get(key); !ok {
					missing.Store(true)
				}
			}
		}(worker)
	}

	close(start)
	wg.Wait()

	if missing.Load() {
		t.Fatal("get missed a key after merge")
	}

	expectedSize := 0
	for keyID := 0; keyID < keys; keyID++ {
		want := 0
		for worker := 0; worker < workers; worker++ {
			want += local[worker][keyID]
		}
		if want == 0 {
			continue
		}
		expectedSize++
		got, ok := m.Get(testKey(keyID))
		if !ok || got != want {
			t.Fatalf("key=%d got=%d ok=%v want=%d", keyID, got, ok, want)
		}
	}
	if got := m.Size(); got != expectedSize {
		t.Fatalf("size got=%d want=%d", got, expectedSize)
	}
}

func TestConcurrentMixedReadWrite(t *testing.T) {
	const (
		workers = 8
		keys    = 256
		ops     = 6000
	)

	m := NewStringMap[int](16)
	for i := 0; i < keys; i++ {
		m.Put(testKey(i), i)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	var failed atomic.Bool

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(500 + id)))
			<-start
			for i := 0; i < ops; i++ {
				key := testKey(rng.Intn(keys))
				if i&7 == 0 {
					m.Put(key, id+i)
					continue
				}
				if _, ok := m.Get(key); !ok {
					failed.Store(true)
				}
			}
		}(worker)
	}

	close(start)
	wg.Wait()

	if failed.Load() {
		t.Fatal("reader observed a missing hot key")
	}
}
