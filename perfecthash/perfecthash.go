package perfecthash

import "sort"

const (
	FNV64OFFSET = 0xcbf29ce484222325
	FNV64PRIME  = 0x100000001b3
	MAXD        = 1 << 24
)

type Index struct {
	displacements []uint32
	keys          []string
	values        []string
	numBuckets    int
	tableSize     int
}

func mix(h uint64) uint64 {
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

func hashKey(s string, seed uint64) uint64 {
	h := seed + FNV64OFFSET
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= FNV64PRIME
	}
	return mix(h)
}

func fillSlots(keys []string, bucket []int, d uint32, tableSize int, table []int, slots []int) bool {
	for i, keyIdx := range bucket {
		slot := int(hashKey(keys[keyIdx], uint64(d)) % uint64(tableSize))
		if table[slot] >= 0 {
			return false
		}
		for j := 0; j < i; j++ {
			if slots[j] == slot {
				return false
			}
		}
		slots[i] = slot
	}
	return true
}

func Build(keys, values []string) *Index {
	n := len(keys)
	if n == 0 {
		return &Index{}
	}
	tableSize := n + n/4 + 2
	numBuckets := n
	buckets := make([][]int, numBuckets)
	widest := 0
	for i, key := range keys {
		b := int(hashKey(key, 0) % uint64(numBuckets))
		buckets[b] = append(buckets[b], i)
		if w := len(buckets[b]); w > widest {
			widest = w
		}
	}
	order := make([]int, numBuckets)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		return len(buckets[order[a]]) > len(buckets[order[b]])
	})
	table := make([]int, tableSize)
	for i := range table {
		table[i] = -1
	}
	displacements := make([]uint32, numBuckets)
	slots := make([]int, widest)
	for _, bi := range order {
		bucket := buckets[bi]
		if len(bucket) == 0 {
			continue
		}
		s := slots[:len(bucket)]
		var placed bool
		for d := uint32(1); d < MAXD; d++ {
			if !fillSlots(keys, bucket, d, tableSize, table, s) {
				continue
			}
			for k, slot := range s {
				table[slot] = bucket[k]
			}
			displacements[bi] = d
			placed = true
			break
		}
		if !placed {
			panic("perfecthash: cannot place bucket")
		}
	}
	outK, outV := make([]string, tableSize), make([]string, tableSize)
	for slot, ki := range table {
		if ki >= 0 {
			outK[slot] = keys[ki]
			outV[slot] = values[ki]
		}
	}
	return &Index{displacements, outK, outV, numBuckets, tableSize}
}

func (idx *Index) Get(key string) (string, bool) {
	if idx.tableSize == 0 {
		return "", false
	}
	b := int(hashKey(key, 0) % uint64(idx.numBuckets))
	d := idx.displacements[b]
	if d == 0 {
		return "", false
	}
	slot := int(hashKey(key, uint64(d)) % uint64(idx.tableSize))
	if idx.keys[slot] == key {
		return idx.values[slot], true
	}
	return "", false
}
