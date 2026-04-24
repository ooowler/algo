package perfecthash

import "sort"

const (
	FNV64OFFSET = 0xcbf29ce484222325
	FNV64PRIME  = 0x100000001b3

	maxDisplacement = 1 << 24
	singletonMask   = uint32(1 << 31)
	slotSeedSalt    = 0x9e3779b97f4a7c15
)

type Index struct {
	displacements []uint32
	keys          []string
	values        []string
	numBuckets    int
	tableSize     int
	n             int
	seed          uint64
	slotSeed      uint64
}

type buildAttempt struct {
	tableSize int
	seed      uint64
}

func mix(h uint64) uint64 {
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

func hashKey(s string) uint64 {
	h := uint64(FNV64OFFSET)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= FNV64PRIME
	}
	return mix(h)
}

func bucketHash(hash, seed uint64) uint64 {
	return mix(hash ^ seed)
}

func slotHash(hash, slotSeed uint64, displacement uint32) uint64 {
	return mix(hash ^ slotSeed ^ mix(uint64(displacement)+slotSeedSalt))
}

func Build(keys, values []string) *Index {
	if len(keys) != len(values) {
		panic("perfecthash: mismatched keys and values")
	}
	n := len(keys)
	if n == 0 {
		return &Index{}
	}
	hashes := make([]uint64, n)
	for i, key := range keys {
		hashes[i] = hashKey(key)
	}
	for _, attempt := range buildAttempts(n) {
		if idx, ok := tryBuild(keys, values, hashes, attempt); ok {
			return idx
		}
	}
	panic("perfecthash: cannot build index")
}

func buildAttempts(n int) []buildAttempt {
	base := []int{0, n / 16, n / 8, n / 4}
	out := make([]buildAttempt, 0, len(base)*6)
	for _, extra := range base {
		tableSize := n + extra
		if tableSize < n {
			tableSize = n
		}
		for i := 0; i < 6; i++ {
			seed := mix(uint64(i+1) * 0x9e3779b97f4a7c15)
			out = append(out, buildAttempt{tableSize: tableSize, seed: seed})
		}
	}
	return out
}

func tryBuild(keys, values []string, hashes []uint64, attempt buildAttempt) (*Index, bool) {
	n := len(keys)
	numBuckets := n
	counts := make([]int, numBuckets)
	widest := 0
	for _, hash := range hashes {
		bucket := int(bucketHash(hash, attempt.seed) % uint64(numBuckets))
		counts[bucket]++
		if counts[bucket] > widest {
			widest = counts[bucket]
		}
	}

	offsets := make([]int, numBuckets+1)
	for i := 0; i < numBuckets; i++ {
		offsets[i+1] = offsets[i] + counts[i]
	}
	entries := make([]int, n)
	cursor := make([]int, numBuckets)
	copy(cursor, offsets[:numBuckets])
	for i, hash := range hashes {
		bucket := int(bucketHash(hash, attempt.seed) % uint64(numBuckets))
		pos := cursor[bucket]
		entries[pos] = i
		cursor[bucket] = pos + 1
	}

	order := make([]int, 0, numBuckets)
	for i, count := range counts {
		if count > 0 {
			order = append(order, i)
		}
	}
	sort.Slice(order, func(i, j int) bool {
		return counts[order[i]] > counts[order[j]]
	})

	table := make([]int, attempt.tableSize)
	for i := range table {
		table[i] = -1
	}
	displacements := make([]uint32, numBuckets)
	marks := make([]int, attempt.tableSize)
	slots := make([]int, widest)
	slotSeed := mix(attempt.seed + slotSeedSalt)
	markID := 1
	singletons := make([]int, 0, n)

	for _, bucketIndex := range order {
		start, end := offsets[bucketIndex], offsets[bucketIndex+1]
		bucket := entries[start:end]
		switch len(bucket) {
		case 0:
			continue
		case 1:
			singletons = append(singletons, bucketIndex)
		default:
			d, ok := placeBucket(hashes, bucket, slotSeed, table, marks, &markID, slots[:len(bucket)])
			if !ok {
				return nil, false
			}
			displacements[bucketIndex] = d
		}
	}

	freeSlots := make([]int, 0, attempt.tableSize)
	for slot, keyIndex := range table {
		if keyIndex < 0 {
			freeSlots = append(freeSlots, slot)
		}
	}
	if len(freeSlots) < len(singletons) {
		return nil, false
	}
	for i, bucketIndex := range singletons {
		slot := freeSlots[i]
		keyIndex := entries[offsets[bucketIndex]]
		table[slot] = keyIndex
		displacements[bucketIndex] = singletonMask | uint32(slot)
	}

	outKeys := make([]string, attempt.tableSize)
	outValues := make([]string, attempt.tableSize)
	for slot, keyIndex := range table {
		if keyIndex >= 0 {
			outKeys[slot] = keys[keyIndex]
			outValues[slot] = values[keyIndex]
		}
	}
	return &Index{
		displacements: displacements,
		keys:          outKeys,
		values:        outValues,
		numBuckets:    numBuckets,
		tableSize:     attempt.tableSize,
		n:             n,
		seed:          attempt.seed,
		slotSeed:      slotSeed,
	}, true
}

func placeBucket(
	hashes []uint64,
	bucket []int,
	slotSeed uint64,
	table []int,
	marks []int,
	markID *int,
	slots []int,
) (uint32, bool) {
	for displacement := uint32(1); displacement < maxDisplacement; displacement++ {
		*markID = *markID + 1
		currentMark := *markID
		ok := true
		for i, keyIndex := range bucket {
			slot := int(slotHash(hashes[keyIndex], slotSeed, displacement) % uint64(len(table)))
			if table[slot] >= 0 || marks[slot] == currentMark {
				ok = false
				break
			}
			marks[slot] = currentMark
			slots[i] = slot
		}
		if !ok {
			continue
		}
		for i, slot := range slots {
			table[slot] = bucket[i]
		}
		return displacement, true
	}
	return 0, false
}

func (idx *Index) BitsPerKey() float64 {
	if idx.n == 0 {
		return 0
	}
	return float64(len(idx.displacements)) * 32.0 / float64(idx.n)
}

func (idx *Index) LoadFactor() float64 {
	if idx.tableSize == 0 {
		return 0
	}
	return float64(idx.n) / float64(idx.tableSize)
}

func (idx *Index) N() int { return idx.n }

func (idx *Index) Get(key string) (string, bool) {
	if idx.tableSize == 0 {
		return "", false
	}
	hash := hashKey(key)
	bucket := int(bucketHash(hash, idx.seed) % uint64(idx.numBuckets))
	displacement := idx.displacements[bucket]
	if displacement == 0 {
		return "", false
	}
	slot := 0
	if displacement&singletonMask != 0 {
		slot = int(displacement &^ singletonMask)
	} else {
		slot = int(slotHash(hash, idx.slotSeed, displacement) % uint64(idx.tableSize))
	}
	if idx.keys[slot] == key {
		return idx.values[slot], true
	}
	return "", false
}
