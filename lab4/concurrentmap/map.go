package concurrentmap

import (
	"sync"
	"sync/atomic"
)

const (
	minBucketCount = 64
	defaultMaxLoad = 8.0
)

type Hasher[K comparable] func(K) uint64

type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

type Iterator[K comparable, V any] struct {
	items []Pair[K, V]
	next  int
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

type bucketData[K comparable, V any] struct {
	entries []entry[K, V]
}

type bucket[K comparable, V any] struct {
	mu   sync.Mutex
	data atomic.Pointer[bucketData[K, V]]
}

type table[K comparable, V any] struct {
	buckets []bucket[K, V]
	mask    uint64
}

type Map[K comparable, V any] struct {
	hasher   Hasher[K]
	maxLoad  float64
	size     atomic.Int64
	current  atomic.Pointer[table[K, V]]
	resizeMu sync.RWMutex
}

func New[K comparable, V any](bucketCount int, hasher Hasher[K]) *Map[K, V] {
	if hasher == nil {
		panic("nil hasher")
	}
	m := &Map[K, V]{
		hasher:  hasher,
		maxLoad: defaultMaxLoad,
	}
	m.current.Store(newTable[K, V](bucketCount))
	return m
}

func NewStringMap[V any](bucketCount int) *Map[string, V] {
	return New[string, V](bucketCount, HashString)
}

func HashString(key string) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	hash := uint64(offset64)
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}
	return hash
}

func (m *Map[K, V]) Put(key K, value V) {
	inserted, _ := m.store(key, value, nil)
	if inserted {
		m.maybeGrow()
	}
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	current := m.current.Load()
	idx := int(m.hasher(key) & current.mask)
	data := current.buckets[idx].data.Load()
	if data != nil {
		for _, item := range data.entries {
			if item.key == key {
				return item.value, true
			}
		}
	}
	var zero V
	return zero, false
}

func (m *Map[K, V]) Merge(key K, value V, merger func(V, V) V) V {
	inserted, merged := m.store(key, value, merger)
	if inserted {
		m.maybeGrow()
	}
	return merged
}

func (m *Map[K, V]) Size() int {
	return int(m.size.Load())
}

func (m *Map[K, V]) Clear() {
	m.resizeMu.Lock()
	current := m.current.Load()
	m.current.Store(newTable[K, V](len(current.buckets)))
	m.size.Store(0)
	m.resizeMu.Unlock()
}

func (m *Map[K, V]) Iterator() *Iterator[K, V] {
	current := m.current.Load()
	items := make([]Pair[K, V], 0, m.Size())
	for i := range current.buckets {
		data := current.buckets[i].data.Load()
		if data == nil {
			continue
		}
		for _, item := range data.entries {
			items = append(items, Pair[K, V]{
				Key:   item.key,
				Value: item.value,
			})
		}
	}
	return &Iterator[K, V]{items: items}
}

func (m *Map[K, V]) BucketCount() int {
	return len(m.current.Load().buckets)
}

func (m *Map[K, V]) LoadFactor() float64 {
	return float64(m.Size()) / float64(m.BucketCount())
}

func (it *Iterator[K, V]) Next() (Pair[K, V], bool) {
	if it.next >= len(it.items) {
		var zero Pair[K, V]
		return zero, false
	}
	item := it.items[it.next]
	it.next++
	return item, true
}

func (m *Map[K, V]) store(key K, value V, merger func(V, V) V) (bool, V) {
	m.resizeMu.RLock()
	current := m.current.Load()
	idx := int(m.hasher(key) & current.mask)
	b := &current.buckets[idx]
	b.mu.Lock()

	data := b.data.Load()
	entries := cloneEntries(data)
	for i := range entries {
		if entries[i].key != key {
			continue
		}
		if merger != nil {
			value = merger(entries[i].value, value)
		}
		entries[i].value = value
		b.data.Store(&bucketData[K, V]{entries: entries})
		b.mu.Unlock()
		m.resizeMu.RUnlock()
		return false, value
	}

	entries = append(entries, entry[K, V]{key: key, value: value})
	b.data.Store(&bucketData[K, V]{entries: entries})
	b.mu.Unlock()
	m.size.Add(1)
	m.resizeMu.RUnlock()
	return true, value
}

func (m *Map[K, V]) maybeGrow() {
	current := m.current.Load()
	if float64(m.Size()) <= float64(len(current.buckets))*m.maxLoad {
		return
	}
	m.resize(len(current.buckets) << 1)
}

func (m *Map[K, V]) resize(bucketCount int) {
	m.resizeMu.Lock()
	defer m.resizeMu.Unlock()

	current := m.current.Load()
	if float64(m.Size()) <= float64(len(current.buckets))*m.maxLoad {
		return
	}

	next := newTable[K, V](bucketCount)
	chains := make([][]entry[K, V], len(next.buckets))
	for i := range current.buckets {
		data := current.buckets[i].data.Load()
		if data == nil {
			continue
		}
		for _, item := range data.entries {
			idx := int(m.hasher(item.key) & next.mask)
			chains[idx] = append(chains[idx], item)
		}
	}
	for i := range chains {
		if len(chains[i]) == 0 {
			continue
		}
		next.buckets[i].data.Store(&bucketData[K, V]{entries: chains[i]})
	}
	m.current.Store(next)
}

func cloneEntries[K comparable, V any](data *bucketData[K, V]) []entry[K, V] {
	if data == nil || len(data.entries) == 0 {
		return nil
	}
	out := make([]entry[K, V], len(data.entries))
	copy(out, data.entries)
	return out
}

func newTable[K comparable, V any](bucketCount int) *table[K, V] {
	count := nextPow2(bucketCount)
	if count < minBucketCount {
		count = minBucketCount
	}
	return &table[K, V]{
		buckets: make([]bucket[K, V], count),
		mask:    uint64(count - 1),
	}
}

func nextPow2(v int) int {
	if v <= 1 {
		return 1
	}
	n := 1
	for n < v {
		n <<= 1
	}
	return n
}
