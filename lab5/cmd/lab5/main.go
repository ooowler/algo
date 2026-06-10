package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"lab5/search"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = runBuild(os.Args[2:])
	case "search":
		err = runSearch(os.Args[2:])
	case "demo":
		err = runDemo(os.Args[2:])
	case "sample":
		err = runSample(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  lab5 build  -docs data/docs.jsonl|dir -index data/index.lab5")
	fmt.Fprintln(os.Stderr, "  lab5 search -index data/index.lab5 -q 'history AND NOT (russia AND china)' [-limit 10]")
	fmt.Fprintln(os.Stderr, "  lab5 demo   -index data/demo.lab5")
	fmt.Fprintln(os.Stderr, "  lab5 sample -out data/sample.jsonl")
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	docsPath := fs.String("docs", "", "jsonl file or directory with .txt/.md documents")
	indexPath := fs.String("index", "data/index.lab5", "output index file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *docsPath == "" {
		return fmt.Errorf("-docs is required")
	}
	docs, err := search.LoadDocuments(*docsPath)
	if err != nil {
		return err
	}
	idx := search.Build(docs)
	if err := os.MkdirAll(filepath.Dir(*indexPath), 0o755); err != nil {
		return err
	}
	if err := search.WriteIndex(*indexPath, idx); err != nil {
		return err
	}
	fmt.Printf("indexed docs=%d terms=%d raw_postings=%d index=%s\n", len(idx.Docs), len(idx.Terms), idx.RawBytes, *indexPath)
	return nil
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	indexPath := fs.String("index", "data/demo.lab5", "index file")
	query := fs.String("q", "", "query")
	limit := fs.Int("limit", 10, "result limit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *query == "" {
		return fmt.Errorf("-q is required")
	}
	idx, err := search.OpenDiskIndex(*indexPath)
	if err != nil {
		return err
	}
	defer idx.Close()
	results, err := idx.Search(*query, *limit)
	if err != nil {
		return err
	}
	for _, result := range results {
		fmt.Printf("#%d score=%.4f %s\n%s\n\n", result.DocID, result.Score, result.Title, result.Snippet)
	}
	fmt.Println(idx.CompressionStats())
	return nil
}

func runDemo(args []string) error {
	fs := flag.NewFlagSet("demo", flag.ExitOnError)
	indexPath := fs.String("index", "data/demo.lab5", "output index file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	idx := search.Build(search.SampleDocuments())
	if err := os.MkdirAll(filepath.Dir(*indexPath), 0o755); err != nil {
		return err
	}
	if err := search.WriteIndex(*indexPath, idx); err != nil {
		return err
	}
	fmt.Printf("demo index written: %s\n", *indexPath)
	return nil
}

func runSample(args []string) error {
	fs := flag.NewFlagSet("sample", flag.ExitOnError)
	out := fs.String("out", "data/sample.jsonl", "output jsonl")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, doc := range search.SampleDocuments() {
		if err := enc.Encode(doc); err != nil {
			return err
		}
	}
	fmt.Printf("sample written: %s\n", *out)
	return nil
}
