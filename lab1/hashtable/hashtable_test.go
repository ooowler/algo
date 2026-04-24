package hashtable

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "dht-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func mustNew(t *testing.T, dir string, buckets int) *DiskHashTable {
	t.Helper()
	h, err := New(dir, buckets)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func mustSet(t *testing.T, h *DiskHashTable, key, value string) {
	t.Helper()
	if err := h.Set(key, value); err != nil {
		t.Fatal(err)
	}
}

func mustDelete(t *testing.T, h *DiskHashTable, key string) {
	t.Helper()
	if err := h.Delete(key); err != nil {
		t.Fatal(err)
	}
}

func mustGet(t *testing.T, h *DiskHashTable, key string) (string, bool) {
	t.Helper()
	value, ok, err := h.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	return value, ok
}

func TestSetUpdateDeleteAndGet(t *testing.T) {
	h := mustNew(t, tempDir(t), 16)
	mustSet(t, h, "hello", "world")
	value, ok := mustGet(t, h, "hello")
	if !ok || value != "world" {
		t.Fatalf("got %q ok=%v", value, ok)
	}

	mustSet(t, h, "hello", "again")
	value, ok = mustGet(t, h, "hello")
	if !ok || value != "again" {
		t.Fatalf("got %q ok=%v", value, ok)
	}

	mustDelete(t, h, "hello")
	_, ok = mustGet(t, h, "hello")
	if ok {
		t.Fatal("key should be deleted")
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	dir := tempDir(t)
	h := mustNew(t, dir, 8)
	mustSet(t, h, "persist", "yes")
	mustSet(t, h, "other", "value")
	mustDelete(t, h, "other")

	h = mustNew(t, dir, 8)
	value, ok := mustGet(t, h, "persist")
	if !ok || value != "yes" {
		t.Fatalf("expected persisted value, got %q ok=%v", value, ok)
	}
	_, ok = mustGet(t, h, "other")
	if ok {
		t.Fatal("deleted key reappeared after reopen")
	}
}

func TestSingleBucketAndLargeValue(t *testing.T) {
	h := mustNew(t, tempDir(t), 1)
	large := strings.Repeat("payload-", 2048)
	for i := 0; i < 64; i++ {
		mustSet(t, h, strconv.Itoa(i), large+strconv.Itoa(i))
	}
	for i := 0; i < 64; i++ {
		value, ok := mustGet(t, h, strconv.Itoa(i))
		if !ok || value != large+strconv.Itoa(i) {
			t.Fatalf("i=%d got %q ok=%v", i, value, ok)
		}
	}
}

func TestSpecialCharactersInKeyAndValue(t *testing.T) {
	h := mustNew(t, tempDir(t), 16)
	cases := [][2]string{
		{"", ""},
		{"key with spaces", "value with spaces"},
		{"tab\tkey", "tab\tvalue"},
		{"line\nkey", "line\nvalue"},
		{"unicode-ключ", "unicode-значение"},
	}
	for _, tc := range cases {
		mustSet(t, h, tc[0], tc[1])
		value, ok := mustGet(t, h, tc[0])
		if !ok || value != tc[1] {
			t.Fatalf("key %q: got %q ok=%v", tc[0], value, ok)
		}
	}
}

func TestRandomOperationsWithReopen(t *testing.T) {
	dir := tempDir(t)
	h := mustNew(t, dir, 32)
	ref := make(map[string]string)
	rng := rand.New(rand.NewSource(42))

	verify := func(step int) {
		t.Helper()
		for key, want := range ref {
			value, ok := mustGet(t, h, key)
			if !ok || value != want {
				t.Fatalf("step=%d key=%q got=%q ok=%v want=%q", step, key, value, ok, want)
			}
		}
		for i := 0; i < 16; i++ {
			key := fmt.Sprintf("missing-%d", step*16+i)
			if _, ok := ref[key]; ok {
				continue
			}
			if _, ok := mustGet(t, h, key); ok {
				t.Fatalf("step=%d unexpected hit for %q", step, key)
			}
		}
	}

	for step := 0; step < 2500; step++ {
		key := fmt.Sprintf("key-%d", rng.Intn(128))
		switch rng.Intn(4) {
		case 0, 1:
			value := fmt.Sprintf("value-%d-%d", step, rng.Int63())
			mustSet(t, h, key, value)
			ref[key] = value
		case 2:
			mustDelete(t, h, key)
			delete(ref, key)
		default:
			value, ok := mustGet(t, h, key)
			want, wantOK := ref[key]
			if ok != wantOK || value != want {
				t.Fatalf("step=%d key=%q got=%q ok=%v want=%q wantOK=%v", step, key, value, ok, want, wantOK)
			}
		}

		if step%200 == 0 {
			h = mustNew(t, dir, 32)
			verify(step)
		}
	}

	h = mustNew(t, dir, 32)
	verify(2500)
}

func BenchmarkSet(b *testing.B) {
	h, err := New(b.TempDir(), 256)
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()
	keys := makeBenchKeys(0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Set(keys[i], "value")
	}
}

func BenchmarkGet(b *testing.B) {
	h, err := New(b.TempDir(), 256)
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()
	keys := makeBenchKeys(0, 1000)
	for i := 0; i < 1000; i++ {
		_ = h.Set(keys[i], "value")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = h.Get(keys[i%1000])
	}
}

func BenchmarkDelete(b *testing.B) {
	h, err := New(b.TempDir(), 256)
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()
	keys := makeBenchKeys(0, b.N)
	for i := 0; i < b.N; i++ {
		_ = h.Set(keys[i], "value")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Delete(keys[i])
	}
}
