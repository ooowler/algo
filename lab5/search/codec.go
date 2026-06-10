package search

import (
	"encoding/binary"
	"errors"
	"math/bits"
)

const blockSize = 128

func EncodeUint32s(values []uint32) []byte {
	out := make([]byte, 4, 4+len(values)*2)
	binary.LittleEndian.PutUint32(out[:4], uint32(len(values)))
	for start := 0; start < len(values); start += blockSize {
		end := start + blockSize
		if end > len(values) {
			end = len(values)
		}
		width := uint8(0)
		for _, v := range values[start:end] {
			if w := uint8(bits.Len32(v)); w > width {
				width = w
			}
		}
		var header [3]byte
		binary.LittleEndian.PutUint16(header[:2], uint16(end-start))
		header[2] = width
		out = append(out, header[:]...)
		out = packBlock(out, values[start:end], width)
	}
	return out
}

func DecodeUint32s(data []byte) ([]uint32, int, error) {
	if len(data) < 4 {
		return nil, 0, errors.New("short encoded integer sequence")
	}
	total := int(binary.LittleEndian.Uint32(data[:4]))
	values := make([]uint32, 0, total)
	off := 4
	for len(values) < total {
		if len(data)-off < 3 {
			return nil, 0, errors.New("short bitpacking block header")
		}
		count := int(binary.LittleEndian.Uint16(data[off : off+2]))
		width := data[off+2]
		off += 3
		if count <= 0 || count > blockSize || len(values)+count > total || width > 32 {
			return nil, 0, errors.New("bad bitpacking block header")
		}
		need := (count*int(width) + 7) / 8
		if len(data)-off < need {
			return nil, 0, errors.New("short bitpacking block")
		}
		values = unpackBlock(values, data[off:off+need], count, width)
		off += need
	}
	return values, off, nil
}

func packBlock(out []byte, values []uint32, width uint8) []byte {
	if width == 0 {
		return out
	}
	bitPos := 0
	byteCount := (len(values)*int(width) + 7) / 8
	start := len(out)
	out = append(out, make([]byte, byteCount)...)
	for _, v := range values {
		x := v
		for bit := uint8(0); bit < width; bit++ {
			if x&(1<<bit) != 0 {
				out[start+bitPos/8] |= 1 << uint(bitPos%8)
			}
			bitPos++
		}
	}
	return out
}

func unpackBlock(out []uint32, data []byte, count int, width uint8) []uint32 {
	if width == 0 {
		for i := 0; i < count; i++ {
			out = append(out, 0)
		}
		return out
	}
	mask := uint64(1<<width) - 1
	bitPos := 0
	for i := 0; i < count; i++ {
		bytePos := bitPos / 8
		shift := uint(bitPos % 8)
		var buf [8]byte
		copy(buf[:], data[bytePos:])
		out = append(out, uint32((binary.LittleEndian.Uint64(buf[:])>>shift)&mask))
		bitPos += int(width)
	}
	return out
}

func encodePostings(list PostingList) []byte {
	docGaps := make([]uint32, 0, len(list.Items))
	tfs := make([]uint32, 0, len(list.Items))
	positions := make([]uint32, 0)
	prevDoc := uint32(0)
	for _, item := range list.Items {
		docGaps = append(docGaps, item.DocID-prevDoc)
		prevDoc = item.DocID
		tfs = append(tfs, item.TF)
		prevPos := uint32(0)
		for _, pos := range item.Positions {
			positions = append(positions, pos-prevPos)
			prevPos = pos
		}
	}
	out := EncodeUint32s(docGaps)
	out = append(out, EncodeUint32s(tfs)...)
	out = append(out, EncodeUint32s(positions)...)
	return out
}

func decodePostings(data []byte) (PostingList, error) {
	docGaps, n, err := DecodeUint32s(data)
	if err != nil {
		return PostingList{}, err
	}
	tfs, m, err := DecodeUint32s(data[n:])
	if err != nil {
		return PostingList{}, err
	}
	posDeltas, _, err := DecodeUint32s(data[n+m:])
	if err != nil {
		return PostingList{}, err
	}
	if len(docGaps) != len(tfs) {
		return PostingList{}, errors.New("posting doc/tf length mismatch")
	}
	items := make([]Posting, 0, len(docGaps))
	docID := uint32(0)
	posOff := 0
	for i := range docGaps {
		docID += docGaps[i]
		tf := tfs[i]
		if posOff+int(tf) > len(posDeltas) {
			return PostingList{}, errors.New("posting position length mismatch")
		}
		positions := make([]uint32, int(tf))
		pos := uint32(0)
		for j := range positions {
			pos += posDeltas[posOff+j]
			positions[j] = pos
		}
		posOff += int(tf)
		items = append(items, Posting{DocID: docID, TF: tf, Positions: positions})
	}
	if posOff != len(posDeltas) {
		return PostingList{}, errors.New("unused position deltas")
	}
	return newPostingList(items), nil
}
