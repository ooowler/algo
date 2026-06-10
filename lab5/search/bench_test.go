package search

import (
	"fmt"
	"path/filepath"
	"testing"
)

var benchQueries = []struct {
	name  string
	query string
}{
	{"AND_balanced", "alpha AND beta"},
	{"OR_balanced", "alpha OR beta"},
	{"NOT_nested", "history AND NOT (russia AND china)"},
	{"ADJ_phrase", "quick ADJ brown"},
	{"NEAR_window", "russia NEAR/3 china"},
	{"Ranking_many", "alpha OR gamma OR omega"},
}

func BenchmarkQueriesMemory(b *testing.B) {
	idx := Build(SyntheticDocuments(25000))
	for _, item := range benchQueries {
		b.Run(item.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = idx.Search(item.query, 20)
			}
		})
	}
}

func BenchmarkQueriesDiskMmap(b *testing.B) {
	mem := Build(SyntheticDocuments(25000))
	path := filepath.Join(b.TempDir(), "index.lab5")
	if err := WriteIndex(path, mem); err != nil {
		b.Fatal(err)
	}
	idx, err := OpenDiskIndex(path)
	if err != nil {
		b.Fatal(err)
	}
	defer idx.Close()
	for _, item := range benchQueries {
		b.Run(item.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = idx.Search(item.query, 20)
			}
		})
	}
}

func BenchmarkCompressionBaseline(b *testing.B) {
	for _, n := range []int{1000, 5000, 25000} {
		b.Run(fmt.Sprintf("Docs%d", n), func(b *testing.B) {
			docs := SyntheticDocuments(n)
			var raw uint64
			var compressed uint64
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := Build(docs)
				raw = idx.RawBytes
				compressed = compressedPostingsBytes(idx)
				if compressed == 0 {
					b.Fatal("empty compressed index")
				}
			}
			b.ReportMetric(float64(raw), "raw_B")
			b.ReportMetric(float64(compressed), "compressed_B")
			b.ReportMetric(float64(compressed)/float64(raw), "ratio")
		})
	}
}

func compressedPostingsBytes(idx *Index) uint64 {
	var compressed uint64
	for _, list := range idx.Terms {
		compressed += uint64(len(encodePostings(list)))
	}
	return compressed
}
