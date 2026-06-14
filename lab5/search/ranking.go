package search

import (
	"math"
	"sort"
	"strings"
)

type Result struct {
	DocID   uint32  `json:"doc_id"`
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
}

type SearchStats struct {
	Results      []Result `json:"results"`
	TotalMatches int      `json:"total_matches"`
	Limit        int      `json:"limit"`
}

type rankedDoc struct {
	DocID uint32
	Score float64
}

func Search(searcher Searcher, query string, limit int) ([]Result, error) {
	stats, err := SearchDetailed(searcher, query, limit)
	if err != nil {
		return nil, err
	}
	return stats.Results, nil
}

func SearchDetailed(searcher Searcher, query string, limit int) (SearchStats, error) {
	node, err := ParseQuery(query)
	if err != nil {
		return SearchStats{}, err
	}
	matches, err := evalNode(searcher, node)
	if err != nil {
		return SearchStats{}, err
	}
	terms := positiveTerms(node)
	scores := bm25Scores(searcher, matches, terms)
	ranked := make([]rankedDoc, 0, len(matches.Items))
	for _, item := range matches.Items {
		ranked = append(ranked, rankedDoc{DocID: item.DocID, Score: scores[item.DocID]})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].DocID < ranked[j].DocID
		}
		return ranked[i].Score > ranked[j].Score
	})
	if limit <= 0 || limit > len(ranked) {
		limit = len(ranked)
	}
	results := make([]Result, 0, limit)
	for _, item := range ranked[:limit] {
		doc, ok := searcher.Doc(item.DocID)
		if !ok {
			continue
		}
		results = append(results, Result{
			DocID:   item.DocID,
			Title:   doc.Title,
			Score:   item.Score,
			Snippet: makeSnippet(doc, terms),
		})
	}
	return SearchStats{
		Results:      results,
		TotalMatches: len(matches.Items),
		Limit:        limit,
	}, nil
}

func positiveTerms(node Node) []string {
	set := make(map[string]struct{})
	node.positiveTerms(false, set)
	terms := make([]string, 0, len(set))
	for term := range set {
		terms = append(terms, term)
	}
	sort.Strings(terms)
	return terms
}

func bm25Scores(searcher Searcher, matches PostingList, terms []string) map[uint32]float64 {
	scores := make(map[uint32]float64, len(matches.Items))
	if len(terms) == 0 || searcher.DocCount() == 0 {
		return scores
	}
	for _, term := range terms {
		list, ok := searcher.Postings(term)
		if !ok {
			continue
		}
		stats, ok := searcher.TermStats(term)
		if !ok || stats.DF == 0 {
			continue
		}
		idf := math.Log(1 + (float64(searcher.DocCount())-float64(stats.DF)+0.5)/(float64(stats.DF)+0.5))
		matchPos := 0
		for _, posting := range list.Items {
			matchPos = advanceTo(matches, matchPos, posting.DocID)
			if matchPos >= len(matches.Items) {
				break
			}
			if matches.Items[matchPos].DocID != posting.DocID {
				continue
			}
			doc, ok := searcher.Doc(posting.DocID)
			if !ok {
				continue
			}
			scores[posting.DocID] += bm25(idf, float64(posting.TF), float64(doc.Length), searcher.AverageDocLength())
		}
	}
	return scores
}

func bm25(idf float64, tf float64, docLen float64, avgDocLen float64) float64 {
	const k1 = 1.2
	const b = 0.75
	if avgDocLen <= 0 {
		avgDocLen = 1
	}
	denom := tf + k1*(1-b+b*docLen/avgDocLen)
	return idf * (tf * (k1 + 1)) / denom
}

func makeSnippet(doc DocInfo, terms []string) string {
	text := strings.TrimSpace(doc.Text)
	if text == "" {
		return ""
	}
	runes := []rune(text)
	lower := []rune(strings.ToLower(text))
	start := 0
	for _, term := range terms {
		if pos := indexRunes(lower, []rune(term)); pos >= 0 {
			start = pos - 60
			if start < 0 {
				start = 0
			}
			break
		}
	}
	end := start + 220
	if end > len(runes) {
		end = len(runes)
	}
	snippet := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func indexRunes(text []rune, term []rune) int {
	if len(term) == 0 || len(term) > len(text) {
		return -1
	}
	for i := 0; i <= len(text)-len(term); i++ {
		if hasRunePrefix(text[i:], term) {
			return i
		}
	}
	return -1
}

func hasRunePrefix(text []rune, term []rune) bool {
	for i := range term {
		if text[i] != term[i] {
			return false
		}
	}
	return true
}
