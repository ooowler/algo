package search

import (
	"math"
	"sort"
)

type Posting struct {
	DocID     uint32
	TF        uint32
	Positions []uint32
}

type PostingList struct {
	Items    []Posting
	SkipStep int
}

type DocInfo struct {
	ID     uint32  `json:"id"`
	Title  string  `json:"title"`
	Text   string  `json:"text"`
	Length uint32  `json:"length"`
	Norm   float64 `json:"norm"`
}

type TermInfo struct {
	DF  uint32 `json:"df"`
	CTF uint32 `json:"ctf"`
}

type Index struct {
	Docs         []DocInfo
	Terms        map[string]PostingList
	Stats        map[string]TermInfo
	docPos       map[uint32]int
	AvgDocLength float64
	RawBytes     uint64
}

type Searcher interface {
	AllDocIDs() []uint32
	Doc(uint32) (DocInfo, bool)
	Postings(string) (PostingList, bool)
	TermStats(string) (TermInfo, bool)
	DocCount() int
	AverageDocLength() float64
	Search(string, int) ([]Result, error)
}

func Build(docs []Document) *Index {
	builders := make(map[string]map[uint32][]uint32)
	infos := make([]DocInfo, 0, len(docs))
	rawBytes := uint64(0)
	for i, doc := range docs {
		id := doc.ID
		if id == 0 {
			id = uint32(i + 1)
		}
		text := doc.Title + " " + doc.Text
		tokens := Tokenize(text)
		infos = append(infos, DocInfo{
			ID:     id,
			Title:  doc.Title,
			Text:   doc.Text,
			Length: uint32(len(tokens)),
		})
		for _, token := range tokens {
			perDoc := builders[token.Term]
			if perDoc == nil {
				perDoc = make(map[uint32][]uint32)
				builders[token.Term] = perDoc
			}
			perDoc[id] = append(perDoc[id], token.Pos)
		}
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].ID < infos[j].ID })

	terms := make(map[string]PostingList, len(builders))
	stats := make(map[string]TermInfo, len(builders))
	for term, perDoc := range builders {
		ids := make([]uint32, 0, len(perDoc))
		for id := range perDoc {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		items := make([]Posting, 0, len(ids))
		ctf := uint32(0)
		for _, id := range ids {
			positions := perDoc[id]
			sort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })
			ctf += uint32(len(positions))
			rawBytes += uint64(8 + 4*len(positions))
			items = append(items, Posting{DocID: id, TF: uint32(len(positions)), Positions: positions})
		}
		terms[term] = newPostingList(items)
		stats[term] = TermInfo{DF: uint32(len(items)), CTF: ctf}
	}

	idx := &Index{
		Docs:     infos,
		Terms:    terms,
		Stats:    stats,
		docPos:   make(map[uint32]int, len(infos)),
		RawBytes: rawBytes,
	}
	totalLength := uint64(0)
	for i := range idx.Docs {
		idx.docPos[idx.Docs[i].ID] = i
		totalLength += uint64(idx.Docs[i].Length)
	}
	if len(idx.Docs) > 0 {
		idx.AvgDocLength = float64(totalLength) / float64(len(idx.Docs))
	}
	return idx
}

func newPostingList(items []Posting) PostingList {
	step := int(math.Sqrt(float64(len(items))))
	if step < 8 {
		step = 8
	}
	return PostingList{Items: items, SkipStep: step}
}

func (idx *Index) AllDocIDs() []uint32 {
	ids := make([]uint32, len(idx.Docs))
	for i := range idx.Docs {
		ids[i] = idx.Docs[i].ID
	}
	return ids
}

func (idx *Index) Doc(id uint32) (DocInfo, bool) {
	pos, ok := idx.docPos[id]
	if !ok {
		return DocInfo{}, false
	}
	return idx.Docs[pos], true
}

func (idx *Index) Postings(term string) (PostingList, bool) {
	list, ok := idx.Terms[NormalizeTerm(term)]
	return list, ok
}

func (idx *Index) TermStats(term string) (TermInfo, bool) {
	info, ok := idx.Stats[NormalizeTerm(term)]
	return info, ok
}

func (idx *Index) DocCount() int {
	return len(idx.Docs)
}

func (idx *Index) AverageDocLength() float64 {
	return idx.AvgDocLength
}

func (idx *Index) Search(query string, limit int) ([]Result, error) {
	return Search(idx, query, limit)
}
