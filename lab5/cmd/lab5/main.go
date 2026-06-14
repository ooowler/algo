package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"lab5/search"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	case "import-libru":
		err = runImportLibru(os.Args[2:])
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
	fmt.Fprintln(os.Stderr, "  lab5 import-libru -out data/war_and_peace.jsonl file1.html file2.html ...")
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

func runImportLibru(args []string) error {
	fs := flag.NewFlagSet("import-libru", flag.ExitOnError)
	out := fs.String("out", "data/war_and_peace.jsonl", "output jsonl")
	chunkRunes := fs.Int("chunk-runes", 900, "target document fragment size in runes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	inputs := fs.Args()
	if len(inputs) == 0 {
		return fmt.Errorf("at least one html file is required")
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 1024*1024)
	defer w.Flush()
	enc := json.NewEncoder(w)
	docID := uint32(1)
	for _, input := range inputs {
		docs, err := importLibruFile(input, docID, *chunkRunes)
		if err != nil {
			return err
		}
		for _, doc := range docs {
			if err := enc.Encode(doc); err != nil {
				return err
			}
		}
		docID += uint32(len(docs))
	}
	fmt.Printf("imported docs=%d out=%s\n", docID-1, *out)
	return nil
}

func importLibruFile(path string, firstID uint32, chunkRunes int) ([]search.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	title := libruTitle(string(data), filepath.Base(path))
	text := cleanLibruHTML(string(data))
	parts := splitText(text, chunkRunes)
	docs := make([]search.Document, 0, len(parts))
	for i, part := range parts {
		docs = append(docs, search.Document{
			ID:    firstID + uint32(i),
			Title: fmt.Sprintf("%s / фрагмент %03d", title, i+1),
			Text:  part,
		})
	}
	return docs, nil
}

func libruTitle(html string, fallback string) string {
	re := regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return fallback
	}
	return strings.TrimSpace(htmlUnescape(stripTags(matches[1])))
}

func cleanLibruHTML(html string) string {
	text := regexp.MustCompile(`(?is)<script.*?</script>`).ReplaceAllString(html, " ")
	text = regexp.MustCompile(`(?is)<style.*?</style>`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(text, " ")
	text = htmlUnescape(text)
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len([]rune(line)) < 40 {
			continue
		}
		if strings.Contains(line, "Lib.ru") || strings.Contains(line, "Комментарии") || strings.Contains(line, "Рейтинги") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func stripTags(text string) string {
	return regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(text, " ")
}

func htmlUnescape(text string) string {
	text = html.UnescapeString(text)
	replacer := strings.NewReplacer("&nbsp;", " ")
	return replacer.Replace(text)
}

func splitText(text string, maxRunes int) []string {
	paragraphs := strings.Split(text, "\n")
	var docs []string
	var b strings.Builder
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(paragraph)
		if len([]rune(b.String())) >= maxRunes {
			docs = append(docs, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		docs = append(docs, b.String())
	}
	return docs
}
