package perfecthash

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
)

func buildSimple(pairs [][2]string) *Index {
	keys := make([]string, len(pairs))
	vals := make([]string, len(pairs))
	for i, p := range pairs {
		keys[i] = p[0]
		vals[i] = p[1]
	}
	return Build(keys, vals)
}

func TestEmpty(t *testing.T) {
	idx := Build(nil, nil)
	_, ok := idx.Get("anything")
	if ok {
		t.Fatal("empty index should return false")
	}
}

func TestSingleKey(t *testing.T) {
	idx := buildSimple([][2]string{{"hello", "world"}})
	v, ok := idx.Get("hello")
	if !ok || v != "world" {
		t.Fatalf("got %q ok=%v", v, ok)
	}
}

func TestMissingKey(t *testing.T) {
	idx := buildSimple([][2]string{{"a", "1"}, {"b", "2"}})
	_, ok := idx.Get("c")
	if ok {
		t.Fatal("missing key should return false")
	}
}

func TestTwoKeys(t *testing.T) {
	idx := buildSimple([][2]string{{"foo", "bar"}, {"baz", "qux"}})
	v, ok := idx.Get("foo")
	if !ok || v != "bar" {
		t.Fatalf("foo: got %q ok=%v", v, ok)
	}
	v, ok = idx.Get("baz")
	if !ok || v != "qux" {
		t.Fatalf("baz: got %q ok=%v", v, ok)
	}
}

func TestAllKeysRetrievable(t *testing.T) {
	pairs := [][2]string{
		{"alpha", "1"}, {"beta", "2"}, {"gamma", "3"},
		{"delta", "4"}, {"epsilon", "5"}, {"zeta", "6"},
		{"eta", "7"}, {"theta", "8"},
	}
	idx := buildSimple(pairs)
	for _, p := range pairs {
		v, ok := idx.Get(p[0])
		if !ok || v != p[1] {
			t.Fatalf("key %q: expected %q got %q ok=%v", p[0], p[1], v, ok)
		}
	}
}

func TestNoFalsePositives(t *testing.T) {
	pairs := [][2]string{{"a", "1"}, {"b", "2"}, {"c", "3"}}
	idx := buildSimple(pairs)
	notIn := []string{"d", "e", "f", "ab", "bc", "ca", "", "A", "B"}
	for _, k := range notIn {
		_, ok := idx.Get(k)
		if ok {
			t.Fatalf("false positive for key %q", k)
		}
	}
}

func TestLargeRandomKeys(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	n := 1000
	keys := make([]string, n)
	vals := make([]string, n)
	seen := make(map[string]bool)
	for i := 0; i < n; i++ {
		for {
			k := fmt.Sprintf("key-%d-%d", rng.Int63(), i)
			if !seen[k] {
				keys[i] = k
				vals[i] = strconv.Itoa(i)
				seen[k] = true
				break
			}
		}
	}
	idx := Build(keys, vals)
	for i, k := range keys {
		v, ok := idx.Get(k)
		if !ok || v != vals[i] {
			t.Fatalf("key %q: expected %q got %q ok=%v", k, vals[i], v, ok)
		}
	}
}

func TestNumericStringKeys(t *testing.T) {
	n := 100
	keys := make([]string, n)
	vals := make([]string, n)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
		vals[i] = strconv.Itoa(i * i)
	}
	idx := Build(keys, vals)
	for i, k := range keys {
		v, ok := idx.Get(k)
		if !ok || v != vals[i] {
			t.Fatalf("key %q: expected %q got %q ok=%v", k, vals[i], v, ok)
		}
	}
}

func TestCollisionPronePrefixes(t *testing.T) {
	pairs := make([][2]string, 50)
	for i := range pairs {
		pairs[i] = [2]string{fmt.Sprintf("prefix-%04d", i), strconv.Itoa(i)}
	}
	idx := buildSimple(pairs)
	for _, p := range pairs {
		v, ok := idx.Get(p[0])
		if !ok || v != p[1] {
			t.Fatalf("key %q: expected %q got %q ok=%v", p[0], p[1], v, ok)
		}
	}
}

func TestUnicodeKeys(t *testing.T) {
	pairs := [][2]string{
		{"ключ", "значение"},
		{"тест", "данные"},
		{"алгоритм", "хэш"},
	}
	idx := buildSimple(pairs)
	for _, p := range pairs {
		v, ok := idx.Get(p[0])
		if !ok || v != p[1] {
			t.Fatalf("key %q: got %q ok=%v", p[0], v, ok)
		}
	}
}

func BenchmarkBuild(b *testing.B) {
	n := 10000
	keys := make([]string, n)
	vals := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("bench-key-%d", i)
		vals[i] = strconv.Itoa(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Build(keys, vals)
	}
}

func BenchmarkGet(b *testing.B) {
	n := 10000
	keys := make([]string, n)
	vals := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("bench-key-%d", i)
		vals[i] = strconv.Itoa(i)
	}
	idx := Build(keys, vals)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Get(keys[i%n])
	}
}
