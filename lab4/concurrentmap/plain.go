package concurrentmap

type plainMap[K comparable, V any] struct {
	hasher  Hasher[K]
	maxLoad float64
	size    int
	buckets [][]entry[K, V]
	mask    uint64
}

func newPlainMap[K comparable, V any](bucketCount int, hasher Hasher[K]) *plainMap[K, V] {
	count := nextPow2(bucketCount)
	if count < minBucketCount {
		count = minBucketCount
	}
	return &plainMap[K, V]{
		hasher:  hasher,
		maxLoad: defaultMaxLoad,
		buckets: make([][]entry[K, V], count),
		mask:    uint64(count - 1),
	}
}

func newPlainStringMap[V any](bucketCount int) *plainMap[string, V] {
	return newPlainMap[string, V](bucketCount, HashString)
}

func (m *plainMap[K, V]) Get(key K) (V, bool) {
	for _, item := range m.buckets[int(m.hasher(key)&m.mask)] {
		if item.key == key {
			return item.value, true
		}
	}
	var zero V
	return zero, false
}

func (m *plainMap[K, V]) Put(key K, value V) {
	idx := int(m.hasher(key) & m.mask)
	chain := m.buckets[idx]
	for i := range chain {
		if chain[i].key == key {
			chain[i].value = value
			m.buckets[idx] = chain
			return
		}
	}
	m.buckets[idx] = append(chain, entry[K, V]{key: key, value: value})
	m.size++
	if float64(m.size) > float64(len(m.buckets))*m.maxLoad {
		m.resize(len(m.buckets) << 1)
	}
}

func (m *plainMap[K, V]) Merge(key K, value V, merger func(V, V) V) V {
	idx := int(m.hasher(key) & m.mask)
	chain := m.buckets[idx]
	for i := range chain {
		if chain[i].key == key {
			value = merger(chain[i].value, value)
			chain[i].value = value
			m.buckets[idx] = chain
			return value
		}
	}
	m.buckets[idx] = append(chain, entry[K, V]{key: key, value: value})
	m.size++
	if float64(m.size) > float64(len(m.buckets))*m.maxLoad {
		m.resize(len(m.buckets) << 1)
	}
	return value
}

func (m *plainMap[K, V]) Clear() {
	for i := range m.buckets {
		m.buckets[i] = nil
	}
	m.size = 0
}

func (m *plainMap[K, V]) Size() int {
	return m.size
}

func (m *plainMap[K, V]) resize(bucketCount int) {
	next := newPlainMap[K, V](bucketCount, m.hasher)
	for i := range m.buckets {
		for _, item := range m.buckets[i] {
			idx := int(next.hasher(item.key) & next.mask)
			next.buckets[idx] = append(next.buckets[idx], item)
		}
	}
	next.size = m.size
	*m = *next
}
