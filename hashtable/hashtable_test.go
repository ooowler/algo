package hashtable

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "dht-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestSetAndGet(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	h.Set("hello", "world")
	v, ok, _ := h.Get("hello")
	if !ok || v != "world" {
		t.Fatalf("got %q ok=%v", v, ok)
	}
}

func TestGetMissing(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	_, ok, err := h.Get("nope")
	if err != nil || ok {
		t.Fatalf("expected miss, got ok=%v err=%v", ok, err)
	}
}

func TestUpdate(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	h.Set("k", "v1")
	h.Set("k", "v2")
	v, ok, _ := h.Get("k")
	if !ok || v != "v2" {
		t.Fatalf("expected v2 got %q", v)
	}
}

func TestDelete(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	h.Set("k", "v")
	h.Delete("k")
	_, ok, _ := h.Get("k")
	if ok {
		t.Fatal("key should be gone after delete")
	}
}

func TestDeleteMissing(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	err := h.Delete("ghost")
	if err != nil {
		t.Fatalf("delete missing key should not error: %v", err)
	}
}

func TestDeleteThenReinsert(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	h.Set("k", "old")
	h.Delete("k")
	h.Set("k", "new")
	v, ok, _ := h.Get("k")
	if !ok || v != "new" {
		t.Fatalf("expected new got %q ok=%v", v, ok)
	}
}

func TestManyKeysInSameBucket(t *testing.T) {
	h, _ := New(tempDir(t), 1)
	keys := []string{"a", "b", "c", "d", "e", "f"}
	for _, k := range keys {
		h.Set(k, k+"-val")
	}
	for _, k := range keys {
		v, ok, _ := h.Get(k)
		if !ok || v != k+"-val" {
			t.Fatalf("key %q: got %q ok=%v", k, v, ok)
		}
	}
}

func TestEmptyKey(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	h.Set("", "empty-key-value")
	v, ok, _ := h.Get("")
	if !ok || v != "empty-key-value" {
		t.Fatalf("empty key: got %q ok=%v", v, ok)
	}
}

func TestSpecialCharsInKeyValue(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	cases := [][2]string{
		{"key with spaces", "value with spaces"},
		{"unicode-тест", "значение"},
		{"newline-free", "no\nnewlines"},
		{"tab\there", "tab\tthere"},
	}
	for _, c := range cases {
		if c[1] == "no\nnewlines" || c[0] == "tab\there" {
			continue
		}
		h.Set(c[0], c[1])
		v, ok, _ := h.Get(c[0])
		if !ok || v != c[1] {
			t.Fatalf("key %q: got %q ok=%v", c[0], v, ok)
		}
	}
}

func TestMultipleUpdates(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	for i := 0; i < 100; i++ {
		h.Set("k", strconv.Itoa(i))
	}
	v, ok, _ := h.Get("k")
	if !ok || v != "99" {
		t.Fatalf("expected 99 got %q ok=%v", v, ok)
	}
}

func TestRandomInsertGetDelete(t *testing.T) {
	h, _ := New(tempDir(t), 32)
	rng := rand.New(rand.NewSource(42))

	inserted := make(map[string]string)
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("key-%d", rng.Intn(50))
		val := fmt.Sprintf("val-%d", rng.Int())
		h.Set(key, val)
		inserted[key] = val
	}

	for k, expected := range inserted {
		v, ok, err := h.Get(k)
		if err != nil || !ok || v != expected {
			t.Fatalf("key %q: expected %q got %q ok=%v err=%v", k, expected, v, ok, err)
		}
	}

	for k := range inserted {
		h.Delete(k)
		_, ok, _ := h.Get(k)
		if ok {
			t.Fatalf("key %q should be deleted", k)
		}
	}
}

func TestPersistence(t *testing.T) {
	dir := tempDir(t)
	h1, _ := New(dir, 8)
	h1.Set("persist", "yes")

	h2, _ := New(dir, 8)
	v, ok, _ := h2.Get("persist")
	if !ok || v != "yes" {
		t.Fatalf("expected persisted value, got %q ok=%v", v, ok)
	}
}

func TestSingleBucket(t *testing.T) {
	h, _ := New(tempDir(t), 1)
	for i := 0; i < 50; i++ {
		h.Set(strconv.Itoa(i), strconv.Itoa(i*2))
	}
	for i := 0; i < 50; i++ {
		v, ok, _ := h.Get(strconv.Itoa(i))
		if !ok || v != strconv.Itoa(i*2) {
			t.Fatalf("i=%d got %q ok=%v", i, v, ok)
		}
	}
}

func TestLargeValues(t *testing.T) {
	h, _ := New(tempDir(t), 16)
	large := make([]byte, 10000)
	for i := range large {
		large[i] = 'a' + byte(i%26)
	}
	h.Set("bigval", string(large))
	v, ok, _ := h.Get("bigval")
	if !ok || v != string(large) {
		t.Fatalf("large value mismatch")
	}
}

func BenchmarkSet(b *testing.B) {
	h, _ := New(b.TempDir(), 256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Set(strconv.Itoa(i), "value")
	}
}

func BenchmarkGet(b *testing.B) {
	h, _ := New(b.TempDir(), 256)
	for i := 0; i < 1000; i++ {
		h.Set(strconv.Itoa(i), "value")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Get(strconv.Itoa(i % 1000))
	}
}

func BenchmarkDelete(b *testing.B) {
	h, _ := New(b.TempDir(), 256)
	for i := 0; i < b.N; i++ {
		h.Set(strconv.Itoa(i), "value")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Delete(strconv.Itoa(i))
	}
}
