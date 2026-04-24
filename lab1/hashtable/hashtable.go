package hashtable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	recordSet    byte = 1
	recordDelete byte = 2

	recordHeaderSize      = 9
	bucketCompactMinTotal = 64

	fnvOffset = 2166136261
	fnvPrime  = 16777619
)

type bucketState struct {
	mu     sync.RWMutex
	loaded bool
	data   map[string]string
}

type bucketMeta struct {
	path         string
	liveRecords  int
	staleRecords int
	appendFile   *os.File
}

type DiskHashTable struct {
	dir        string
	buckets    []bucketState
	meta       []bucketMeta
	bucketMask uint32
	bufPool    sync.Pool
}

func New(dir string, numBuckets int) (*DiskHashTable, error) {
	if numBuckets <= 0 {
		return nil, fmt.Errorf("numBuckets must be positive")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	h := &DiskHashTable{
		dir:     dir,
		buckets: make([]bucketState, numBuckets),
		meta:    make([]bucketMeta, numBuckets),
	}
	if numBuckets > 0 && numBuckets&(numBuckets-1) == 0 {
		h.bucketMask = uint32(numBuckets - 1)
	}
	for i := range h.buckets {
		h.meta[i].path = filepath.Join(dir, fmt.Sprintf("bucket_%d.dat", i))
	}
	h.bufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	return h, nil
}

func fnv32(s string) uint32 {
	h := uint32(fnvOffset)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= uint32(fnvPrime)
	}
	return h
}

func (h *DiskHashTable) bucketIndex(key string) int {
	hash := fnv32(key)
	if h.bucketMask != 0 {
		return int(hash & h.bucketMask)
	}
	return int(hash % uint32(len(h.buckets)))
}

func (h *DiskHashTable) bucket(key string) (*bucketState, *bucketMeta) {
	idx := h.bucketIndex(key)
	return &h.buckets[idx], &h.meta[idx]
}

func (h *DiskHashTable) Set(key, value string) error {
	b, meta := h.bucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureLoaded(meta); err != nil {
		return err
	}
	if err := meta.appendRecord(&h.bufPool, recordSet, key, value); err != nil {
		return err
	}
	if _, exists := b.data[key]; exists {
		meta.staleRecords++
	} else {
		meta.liveRecords++
	}
	b.data[key] = value
	return compactBucketIfNeeded(&h.bufPool, b, meta)
}

func (h *DiskHashTable) Get(key string) (string, bool, error) {
	b, meta := h.bucket(key)
	b.mu.RLock()
	if b.loaded {
		v, ok := b.data[key]
		b.mu.RUnlock()
		return v, ok, nil
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureLoaded(meta); err != nil {
		return "", false, err
	}
	v, ok := b.data[key]
	return v, ok, nil
}

func (h *DiskHashTable) Delete(key string) error {
	b, meta := h.bucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureLoaded(meta); err != nil {
		return err
	}
	if _, ok := b.data[key]; !ok {
		return nil
	}
	if err := meta.appendRecord(&h.bufPool, recordDelete, key, ""); err != nil {
		return err
	}
	delete(b.data, key)
	meta.liveRecords--
	meta.staleRecords++
	return compactBucketIfNeeded(&h.bufPool, b, meta)
}

func (h *DiskHashTable) Close() error {
	var firstErr error
	for i := range h.buckets {
		b := &h.buckets[i]
		b.mu.Lock()
		if err := h.meta[i].closeAppendFile(); err != nil && firstErr == nil {
			firstErr = err
		}
		b.mu.Unlock()
	}
	return firstErr
}

func (b *bucketState) ensureLoaded(meta *bucketMeta) error {
	if b.loaded {
		return nil
	}
	data, totalRecords, err := loadBucketFile(meta.path)
	if err != nil {
		return err
	}
	b.data = data
	meta.liveRecords = len(data)
	meta.staleRecords = totalRecords - meta.liveRecords
	b.loaded = true
	return nil
}

func loadBucketFile(path string) (map[string]string, int, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]string), 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	data := make(map[string]string)
	totalRecords := 0
	for off := 0; off < len(raw); {
		if len(raw)-off < recordHeaderSize {
			return nil, 0, io.ErrUnexpectedEOF
		}
		op := raw[off]
		off++
		keyLen := int(binary.LittleEndian.Uint32(raw[off : off+4]))
		off += 4
		valLen := int(binary.LittleEndian.Uint32(raw[off : off+4]))
		off += 4
		if keyLen < 0 || valLen < 0 || len(raw)-off < keyLen+valLen {
			return nil, 0, io.ErrUnexpectedEOF
		}
		key := string(raw[off : off+keyLen])
		off += keyLen
		value := string(raw[off : off+valLen])
		off += valLen
		totalRecords++
		switch op {
		case recordSet:
			data[key] = value
		case recordDelete:
			delete(data, key)
		default:
			return nil, 0, fmt.Errorf("unknown record op %d", op)
		}
	}
	return data, totalRecords, nil
}

func (meta *bucketMeta) appendRecord(pool *sync.Pool, op byte, key, value string) error {
	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer pool.Put(buf)
	buf.Grow(recordHeaderSize + len(key) + len(value))
	buf.WriteByte(op)
	var hdr [8]byte
	binary.LittleEndian.PutUint32(hdr[0:4], uint32(len(key)))
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(value)))
	buf.Write(hdr[:])
	buf.WriteString(key)
	buf.WriteString(value)

	f, err := meta.appendHandle()
	if err != nil {
		return err
	}
	_, err = f.Write(buf.Bytes())
	return err
}

func (meta *bucketMeta) appendHandle() (*os.File, error) {
	if meta.appendFile != nil {
		return meta.appendFile, nil
	}
	f, err := os.OpenFile(meta.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	meta.appendFile = f
	return f, nil
}

func (meta *bucketMeta) closeAppendFile() error {
	if meta.appendFile == nil {
		return nil
	}
	err := meta.appendFile.Close()
	meta.appendFile = nil
	return err
}

func compactBucketIfNeeded(pool *sync.Pool, b *bucketState, meta *bucketMeta) error {
	total := meta.liveRecords + meta.staleRecords
	if total < bucketCompactMinTotal || meta.staleRecords <= meta.liveRecords {
		return nil
	}
	if len(b.data) == 0 {
		if err := meta.closeAppendFile(); err != nil {
			return err
		}
		if err := os.Remove(meta.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		meta.staleRecords = 0
		return nil
	}

	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer pool.Put(buf)
	for key, value := range b.data {
		if err := writeSetRecord(buf, key, value); err != nil {
			return err
		}
	}

	tmpPath := meta.path + ".tmp"
	if err := meta.closeAppendFile(); err != nil {
		return err
	}
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, meta.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	meta.staleRecords = 0
	meta.liveRecords = len(b.data)
	return nil
}

func writeSetRecord(buf *bytes.Buffer, key, value string) error {
	if len(key) > int(^uint32(0)) || len(value) > int(^uint32(0)) {
		return fmt.Errorf("record is too large")
	}
	buf.WriteByte(recordSet)
	var hdr [8]byte
	binary.LittleEndian.PutUint32(hdr[0:4], uint32(len(key)))
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(value)))
	buf.Write(hdr[:])
	buf.WriteString(key)
	buf.WriteString(value)
	return nil
}
