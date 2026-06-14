package main

import (
	"bufio"
	"flag"
	"fmt"
	"lab5/search"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"time"
)

type queryCase struct {
	Name  string
	Query string
}

type row struct {
	Name         string
	Query        string
	Min          time.Duration
	Avg          time.Duration
	P95          time.Duration
	Max          time.Duration
	TotalMatches int
	Returned     int
	Error        string
}

type corpusStats struct {
	SourceFiles      int
	SourceLines      int
	WarAndPeaceLines int
	CorpusBytes      int64
	IndexBytes       int64
}

func main() {
	indexPath := flag.String("index", "data/russian_classics.lab5", "mmap index")
	corpusPath := flag.String("corpus", "data/russian_classics.jsonl", "source corpus jsonl")
	sourceDir := flag.String("sources", "data/src_utf8", "source html directory")
	outPath := flag.String("out", "results/stress.md", "markdown report")
	limit := flag.Int("limit", 10, "top K")
	rounds := flag.Int("rounds", 25, "rounds per query")
	cpuProfile := flag.String("cpuprofile", "", "optional cpu profile path")
	flag.Parse()

	idx, err := search.OpenDiskIndex(*indexPath)
	if err != nil {
		exit(err)
	}
	defer idx.Close()

	if *cpuProfile != "" {
		if err := os.MkdirAll(dirName(*cpuProfile), 0o755); err != nil {
			exit(err)
		}
		f, err := os.Create(*cpuProfile)
		if err != nil {
			exit(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			exit(err)
		}
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
		}()
	}

	rows := run(idx, *limit, *rounds)
	if err := os.MkdirAll(dirName(*outPath), 0o755); err != nil {
		exit(err)
	}
	sourceStats := collectCorpusStats(*sourceDir, *corpusPath, *indexPath)
	if err := os.WriteFile(*outPath, []byte(render(idx, *indexPath, *corpusPath, rows, *rounds, sourceStats)), 0o644); err != nil {
		exit(err)
	}
	fmt.Println(*outPath)
}

func run(idx *search.DiskIndex, limit int, rounds int) []row {
	cases := []queryCase{
		{"term_common", "князь"},
		{"term_rare", "ростовы"},
		{"and", "князь AND андрей"},
		{"or", "пьер OR наташа"},
		{"not_nested", "князь AND NOT (француз AND император)"},
		{"adj", "пьер ADJ безухов"},
		{"near_5", "андрей NEAR/5 болконский"},
		{"near_20", "наташа NEAR/20 ростова"},
		{"negative_only", "NOT (князь AND андрей)"},
		{"missing", "несуществующийтерм"},
		{"unicode", "мертвые AND души"},
		{"bad_query", "(князь AND"},
	}
	rows := make([]row, 0, len(cases))
	for _, item := range cases {
		rows = append(rows, runOne(idx, item, limit, rounds))
	}
	return rows
}

func runOne(idx *search.DiskIndex, item queryCase, limit int, rounds int) row {
	times := make([]time.Duration, 0, rounds)
	result := row{Name: item.Name, Query: item.Query}
	for i := 0; i < rounds; i++ {
		start := time.Now()
		stats, err := idx.SearchDetailed(item.Query, limit)
		elapsed := time.Since(start)
		if err != nil {
			result.Error = err.Error()
			break
		}
		result.TotalMatches = stats.TotalMatches
		result.Returned = len(stats.Results)
		times = append(times, elapsed)
	}
	if len(times) == 0 {
		return result
	}
	sort.Slice(times, func(i int, j int) bool { return times[i] < times[j] })
	var sum time.Duration
	for _, value := range times {
		sum += value
	}
	result.Min = times[0]
	result.Avg = sum / time.Duration(len(times))
	result.P95 = times[p95Index(len(times))]
	result.Max = times[len(times)-1]
	return result
}

func render(idx *search.DiskIndex, indexPath string, corpusPath string, rows []row, rounds int, corpus corpusStats) string {
	info := idx.Info()
	var b strings.Builder
	b.WriteString("# Lab5 stress report\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n|---|---:|\n"))
	b.WriteString(fmt.Sprintf("| Index | `%s` |\n", indexPath))
	b.WriteString(fmt.Sprintf("| Index file size | %s |\n", bytesLabel(corpus.IndexBytes)))
	b.WriteString(fmt.Sprintf("| Corpus | `%s` |\n", corpusPath))
	b.WriteString(fmt.Sprintf("| Corpus file size | %s |\n", bytesLabel(corpus.CorpusBytes)))
	b.WriteString(fmt.Sprintf("| Source HTML files | %d |\n", corpus.SourceFiles))
	b.WriteString(fmt.Sprintf("| Source HTML lines | %d |\n", corpus.SourceLines))
	b.WriteString(fmt.Sprintf("| War and Peace source lines | %d |\n", corpus.WarAndPeaceLines))
	b.WriteString(fmt.Sprintf("| Docs | %d |\n", info.Docs))
	b.WriteString(fmt.Sprintf("| Terms | %d |\n", info.Terms))
	b.WriteString(fmt.Sprintf("| Avg doc length | %.2f |\n", info.AvgDocLength))
	b.WriteString(fmt.Sprintf("| Raw postings bytes | %d |\n", info.RawPostingsBytes))
	b.WriteString(fmt.Sprintf("| Compressed postings bytes | %d |\n", info.PostingsBytes))
	b.WriteString(fmt.Sprintf("| Compression ratio | %.3f |\n", ratio(info.PostingsBytes, info.RawPostingsBytes)))
	b.WriteString(fmt.Sprintf("| Rounds per query | %d |\n", rounds))
	b.WriteString(fmt.Sprintf("| Go runtime | %s |\n\n", runtime.Version()))
	b.WriteString("| Case | Query | L0 matches | Returned | min ms | avg ms | p95 ms | max ms | ~QPS | Error |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, item := range rows {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | %d | %.3f | %.3f | %.3f | %.3f | %.0f | %s |\n",
			item.Name,
			escapePipe(item.Query),
			item.TotalMatches,
			item.Returned,
			ms(item.Min),
			ms(item.Avg),
			ms(item.P95),
			ms(item.Max),
			qps(item.Avg),
			escapePipe(item.Error),
		))
	}
	return b.String()
}

func ratio(a uint64, b uint64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func ms(value time.Duration) float64 {
	return float64(value.Microseconds()) / 1000
}

func qps(value time.Duration) float64 {
	if value <= 0 {
		return 0
	}
	return float64(time.Second) / float64(value)
}

func p95Index(n int) int {
	if n <= 1 {
		return 0
	}
	pos := (n*95 + 99) / 100
	return pos - 1
}

func collectCorpusStats(sourceDir string, corpusPath string, indexPath string) corpusStats {
	stats := corpusStats{
		CorpusBytes: fileSize(corpusPath),
		IndexBytes:  fileSize(indexPath),
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return stats
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".html" {
			continue
		}
		path := filepath.Join(sourceDir, entry.Name())
		lines := countLines(path)
		stats.SourceFiles++
		stats.SourceLines += lines
		if strings.HasPrefix(entry.Name(), "tolstoy_war_") {
			stats.WarAndPeaceLines += lines
		}
	}
	return stats
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func bytesLabel(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	mib := float64(size) / 1024 / 1024
	return fmt.Sprintf("%d bytes / %.2f MiB", size, mib)
}

func escapePipe(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func dirName(path string) string {
	pos := strings.LastIndex(path, "/")
	if pos < 0 {
		return "."
	}
	return path[:pos]
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
