package search

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Document struct {
	ID    uint32 `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

func LoadDocuments(path string) ([]Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return loadDocumentDir(path)
	}
	return loadJSONL(path)
}

func loadJSONL(path string) ([]Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var docs []Document
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var doc Document
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if doc.Title == "" {
			doc.Title = fmt.Sprintf("doc-%d", len(docs)+1)
		}
		docs = append(docs, doc)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	assignIDs(docs)
	return docs, nil
}

func loadDocumentDir(root string) ([]Document, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".txt" || ext == ".md" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, errors.New("no .txt or .md documents found")
	}

	docs := make([]Document, 0, len(paths))
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		docs = append(docs, Document{
			ID:    uint32(i + 1),
			Title: title,
			Text:  string(data),
		})
	}
	return docs, nil
}

func assignIDs(docs []Document) {
	used := make(map[uint32]struct{}, len(docs))
	for i := range docs {
		if docs[i].ID == 0 {
			docs[i].ID = uint32(i + 1)
		}
		used[docs[i].ID] = struct{}{}
	}
	next := uint32(1)
	for i := range docs {
		if docs[i].ID != 0 {
			continue
		}
		for {
			if _, ok := used[next]; !ok {
				break
			}
			next++
		}
		docs[i].ID = next
		used[next] = struct{}{}
	}
}
