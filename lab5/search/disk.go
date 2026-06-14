package search

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"syscall"
)

const indexMagic = "LAB5IDX1"

type DiskTerm struct {
	Offset uint64 `json:"offset"`
	Length uint64 `json:"length"`
	DF     uint32 `json:"df"`
	CTF    uint32 `json:"ctf"`
}

type DiskMeta struct {
	Docs             []DocInfo           `json:"docs"`
	Terms            map[string]DiskTerm `json:"terms"`
	AvgDocLength     float64             `json:"avg_doc_length"`
	RawPostingsBytes uint64              `json:"raw_postings_bytes"`
	PostingsBytes    uint64              `json:"postings_bytes"`
}

type DiskIndex struct {
	file   *os.File
	data   []byte
	meta   DiskMeta
	docPos map[uint32]int
}

type IndexInfo struct {
	Docs             int
	Terms            int
	RawPostingsBytes uint64
	PostingsBytes    uint64
	AvgDocLength     float64
}

func WriteIndex(path string, idx *Index) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := make([]byte, 16)
	copy(header, indexMagic)
	if _, err := f.Write(header); err != nil {
		return err
	}

	meta := DiskMeta{
		Docs:             idx.Docs,
		Terms:            make(map[string]DiskTerm, len(idx.Terms)),
		AvgDocLength:     idx.AvgDocLength,
		RawPostingsBytes: idx.RawBytes,
	}
	terms := make([]string, 0, len(idx.Terms))
	for term := range idx.Terms {
		terms = append(terms, term)
	}
	sort.Strings(terms)
	for _, term := range terms {
		off, err := f.Seek(0, 1)
		if err != nil {
			return err
		}
		encoded := encodePostings(idx.Terms[term])
		if _, err := f.Write(encoded); err != nil {
			return err
		}
		stats := idx.Stats[term]
		meta.Terms[term] = DiskTerm{
			Offset: uint64(off),
			Length: uint64(len(encoded)),
			DF:     stats.DF,
			CTF:    stats.CTF,
		}
		meta.PostingsBytes += uint64(len(encoded))
	}

	metaOffset, err := f.Seek(0, 1)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if _, err := f.Write(payload); err != nil {
		return err
	}
	var offBuf [8]byte
	binary.LittleEndian.PutUint64(offBuf[:], uint64(metaOffset))
	if _, err := f.WriteAt(offBuf[:], 8); err != nil {
		return err
	}
	return f.Sync()
}

func OpenDiskIndex(path string) (*DiskIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.Size() < 16 {
		f.Close()
		return nil, errors.New("index file is too small")
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(info.Size()), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, err
	}
	if string(data[:8]) != indexMagic {
		syscall.Munmap(data)
		f.Close()
		return nil, errors.New("bad index magic")
	}
	metaOffset := binary.LittleEndian.Uint64(data[8:16])
	if metaOffset < 16 || metaOffset >= uint64(len(data)) {
		syscall.Munmap(data)
		f.Close()
		return nil, errors.New("bad metadata offset")
	}
	var meta DiskMeta
	if err := json.Unmarshal(data[metaOffset:], &meta); err != nil {
		syscall.Munmap(data)
		f.Close()
		return nil, err
	}
	idx := &DiskIndex{
		file:   f,
		data:   data,
		meta:   meta,
		docPos: make(map[uint32]int, len(meta.Docs)),
	}
	for i := range meta.Docs {
		idx.docPos[meta.Docs[i].ID] = i
	}
	return idx, nil
}

func (idx *DiskIndex) Close() error {
	var err error
	if idx.data != nil {
		err = syscall.Munmap(idx.data)
		idx.data = nil
	}
	if idx.file != nil {
		if closeErr := idx.file.Close(); err == nil {
			err = closeErr
		}
		idx.file = nil
	}
	return err
}

func (idx *DiskIndex) AllDocIDs() []uint32 {
	ids := make([]uint32, len(idx.meta.Docs))
	for i := range idx.meta.Docs {
		ids[i] = idx.meta.Docs[i].ID
	}
	return ids
}

func (idx *DiskIndex) Doc(id uint32) (DocInfo, bool) {
	pos, ok := idx.docPos[id]
	if !ok {
		return DocInfo{}, false
	}
	return idx.meta.Docs[pos], true
}

func (idx *DiskIndex) Postings(term string) (PostingList, bool) {
	term = NormalizeTerm(term)
	info, ok := idx.meta.Terms[term]
	if !ok {
		return PostingList{}, false
	}
	end := info.Offset + info.Length
	if end > uint64(len(idx.data)) {
		return PostingList{}, false
	}
	list, err := decodePostings(idx.data[info.Offset:end])
	if err != nil {
		return PostingList{}, false
	}
	return list, true
}

func (idx *DiskIndex) TermStats(term string) (TermInfo, bool) {
	info, ok := idx.meta.Terms[NormalizeTerm(term)]
	if !ok {
		return TermInfo{}, false
	}
	return TermInfo{DF: info.DF, CTF: info.CTF}, true
}

func (idx *DiskIndex) DocCount() int {
	return len(idx.meta.Docs)
}

func (idx *DiskIndex) AverageDocLength() float64 {
	return idx.meta.AvgDocLength
}

func (idx *DiskIndex) Search(query string, limit int) ([]Result, error) {
	return Search(idx, query, limit)
}

func (idx *DiskIndex) SearchDetailed(query string, limit int) (SearchStats, error) {
	return SearchDetailed(idx, query, limit)
}

func (idx *DiskIndex) CompressionStats() string {
	if idx.meta.RawPostingsBytes == 0 {
		return "raw=0 compressed=0"
	}
	ratio := float64(idx.meta.PostingsBytes) / float64(idx.meta.RawPostingsBytes)
	return fmt.Sprintf("raw=%d compressed=%d ratio=%.3f", idx.meta.RawPostingsBytes, idx.meta.PostingsBytes, ratio)
}

func (idx *DiskIndex) Info() IndexInfo {
	return IndexInfo{
		Docs:             len(idx.meta.Docs),
		Terms:            len(idx.meta.Terms),
		RawPostingsBytes: idx.meta.RawPostingsBytes,
		PostingsBytes:    idx.meta.PostingsBytes,
		AvgDocLength:     idx.meta.AvgDocLength,
	}
}
