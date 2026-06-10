package search

import (
	"fmt"
	"strings"
)

func SyntheticDocuments(n int) []Document {
	terms := []string{"alpha", "beta", "gamma", "delta", "omega", "sigma", "lambda"}
	docs := make([]Document, 0, n)
	for i := 0; i < n; i++ {
		parts := []string{
			"document", fmt.Sprintf("number %d", i),
			"search index ranking benchmark",
		}
		for j, term := range terms {
			if (i+j)%(j+2) == 0 {
				parts = append(parts, term, term)
			}
		}
		if i%11 == 0 {
			parts = append(parts, "quick brown fox")
		}
		if i%13 == 0 {
			parts = append(parts, "russia bridge china history")
		}
		if i%17 == 0 {
			parts = append(parts, "history archive trade route")
		}
		if i%19 == 0 {
			parts = append(parts, "coordinate positions near operator")
		}
		docs = append(docs, Document{
			ID:    uint32(i + 1),
			Title: fmt.Sprintf("Synthetic document %d", i+1),
			Text:  strings.Join(parts, " "),
		})
	}
	return docs
}
