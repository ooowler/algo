package search

import (
	"path/filepath"
	"testing"
)

func TestTrickyAndNotRanking(t *testing.T) {
	idx := Build(SampleDocuments())
	results, err := idx.Search("history AND NOT (russia AND china)", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	for _, result := range results {
		if result.DocID == 2 {
			t.Fatalf("doc with both russia and china must be excluded: %+v", result)
		}
		if result.Title == "" || result.Snippet == "" {
			t.Fatalf("UI fields must be filled: %+v", result)
		}
	}
	if results[0].Score <= 0 {
		t.Fatalf("ranking did not use positive terms: %+v", results[0])
	}
}

func TestImplicitAndBeforeNot(t *testing.T) {
	idx := Build(SampleDocuments())
	results, err := idx.Search("history NOT (russia AND china)", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.DocID == 2 {
			t.Fatalf("doc with both russia and china must be excluded: %+v", result)
		}
	}
}

func TestAdjAndNear(t *testing.T) {
	idx := Build(SampleDocuments())
	adj, err := idx.Search("quick ADJ brown", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(adj) != 1 || adj[0].DocID != 7 {
		t.Fatalf("bad ADJ result: %+v", adj)
	}
	near, err := idx.Search("russia NEAR/3 china", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) != 1 || near[0].DocID != 2 {
		t.Fatalf("bad NEAR result: %+v", near)
	}
}

func TestDiskIndexRoundTrip(t *testing.T) {
	idx := Build(SampleDocuments())
	path := filepath.Join(t.TempDir(), "index.lab5")
	if err := WriteIndex(path, idx); err != nil {
		t.Fatal(err)
	}
	disk, err := OpenDiskIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	defer disk.Close()
	results, err := disk.Search("search AND index", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].DocID != 4 {
		t.Fatalf("bad disk result: %+v", results)
	}
}
