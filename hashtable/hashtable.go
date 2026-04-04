package hashtable

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

const (
	tab        = '\t'
	nl         = '\n'
	fnvOffset  = 2166136261
	fnvPrime   = 16777619
)

type DiskHashTable struct {
	dir        string
	numBuckets int
}

func New(dir string, numBuckets int) (*DiskHashTable, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &DiskHashTable{dir: dir, numBuckets: numBuckets}, nil
}

func fnv32(s string) uint32 {
	h := uint32(fnvOffset)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= uint32(fnvPrime)
	}
	return h
}

func (h *DiskHashTable) path(key string) string {
	i := int(fnv32(key) % uint32(h.numBuckets))
	return filepath.Join(h.dir, fmt.Sprintf("bucket_%d.dat", i))
}

func load(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, line := range bytes.Split(raw, []byte{nl}) {
		if len(line) == 0 {
			continue
		}
		if line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if k, v, ok := bytes.Cut(line, []byte{tab}); ok {
			m[string(k)] = string(v)
		}
	}
	return m, nil
}

func save(path string, data map[string]string) error {
	var buf bytes.Buffer
	buf.Grow(len(data) * 48)
	for k, v := range data {
		buf.WriteString(k)
		buf.WriteByte(tab)
		buf.WriteString(v)
		buf.WriteByte(nl)
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func (h *DiskHashTable) Set(key, value string) error {
	p := h.path(key)
	data, err := load(p)
	if err != nil {
		return err
	}
	data[key] = value
	return save(p, data)
}

func (h *DiskHashTable) Get(key string) (string, bool, error) {
	data, err := load(h.path(key))
	if err != nil {
		return "", false, err
	}
	v, ok := data[key]
	return v, ok, nil
}

func (h *DiskHashTable) Delete(key string) error {
	p := h.path(key)
	data, err := load(p)
	if err != nil {
		return err
	}
	if _, ok := data[key]; !ok {
		return nil
	}
	delete(data, key)
	return save(p, data)
}
