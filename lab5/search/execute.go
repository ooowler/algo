package search

import (
	"fmt"
	"sort"
)

func evalNode(searcher Searcher, node Node) (PostingList, error) {
	switch n := node.(type) {
	case TermNode:
		list, ok := searcher.Postings(n.Term)
		if !ok {
			return PostingList{}, nil
		}
		return list, nil
	case NotNode:
		child, err := evalNode(searcher, n.Child)
		if err != nil {
			return PostingList{}, err
		}
		return difference(allDocs(searcher), child), nil
	case BinaryNode:
		return evalBinaryNode(searcher, n)
	default:
		return PostingList{}, fmt.Errorf("unknown query node")
	}
}

func evalBinaryNode(searcher Searcher, node BinaryNode) (PostingList, error) {
	if node.Op == opAnd {
		if neg, ok := node.Right.(NotNode); ok {
			left, err := evalNode(searcher, node.Left)
			if err != nil {
				return PostingList{}, err
			}
			excluded, err := evalNode(searcher, neg.Child)
			if err != nil {
				return PostingList{}, err
			}
			return difference(left, excluded), nil
		}
		if neg, ok := node.Left.(NotNode); ok {
			right, err := evalNode(searcher, node.Right)
			if err != nil {
				return PostingList{}, err
			}
			excluded, err := evalNode(searcher, neg.Child)
			if err != nil {
				return PostingList{}, err
			}
			return difference(right, excluded), nil
		}
	}
	left, err := evalNode(searcher, node.Left)
	if err != nil {
		return PostingList{}, err
	}
	right, err := evalNode(searcher, node.Right)
	if err != nil {
		return PostingList{}, err
	}
	return evalBinary(node, left, right), nil
}

func evalBinary(node BinaryNode, left PostingList, right PostingList) PostingList {
	switch node.Op {
	case opAnd:
		return intersect(left, right)
	case opOr:
		return union(left, right)
	case opAdj:
		return positional(left, right, 1, true)
	case opNear:
		return positional(left, right, node.Window, false)
	default:
		return PostingList{}
	}
}

func allDocs(searcher Searcher) PostingList {
	ids := searcher.AllDocIDs()
	items := make([]Posting, len(ids))
	for i := range ids {
		items[i] = Posting{DocID: ids[i]}
	}
	return newPostingList(items)
}

func intersect(left PostingList, right PostingList) PostingList {
	out := make([]Posting, 0, min(len(left.Items), len(right.Items)))
	i := 0
	j := 0
	for i < len(left.Items) && j < len(right.Items) {
		a := left.Items[i]
		b := right.Items[j]
		switch {
		case a.DocID == b.DocID:
			out = append(out, mergePosting(a, b))
			i++
			j++
		case a.DocID < b.DocID:
			i = advanceTo(left, i, b.DocID)
		default:
			j = advanceTo(right, j, a.DocID)
		}
	}
	return newPostingList(out)
}

func union(left PostingList, right PostingList) PostingList {
	out := make([]Posting, 0, len(left.Items)+len(right.Items))
	i := 0
	j := 0
	for i < len(left.Items) || j < len(right.Items) {
		if i >= len(left.Items) {
			out = append(out, right.Items[j])
			j++
			continue
		}
		if j >= len(right.Items) {
			out = append(out, left.Items[i])
			i++
			continue
		}
		a := left.Items[i]
		b := right.Items[j]
		if a.DocID == b.DocID {
			out = append(out, mergePosting(a, b))
			i++
			j++
			continue
		}
		if a.DocID < b.DocID {
			out = append(out, a)
			i++
		} else {
			out = append(out, b)
			j++
		}
	}
	return newPostingList(out)
}

func difference(left PostingList, right PostingList) PostingList {
	out := make([]Posting, 0, len(left.Items))
	i := 0
	j := 0
	for i < len(left.Items) {
		if j >= len(right.Items) {
			out = append(out, left.Items[i:]...)
			break
		}
		a := left.Items[i]
		b := right.Items[j]
		switch {
		case a.DocID == b.DocID:
			i++
			j++
		case a.DocID < b.DocID:
			out = append(out, a)
			i++
		default:
			j = advanceTo(right, j, a.DocID)
		}
	}
	return newPostingList(out)
}

func positional(left PostingList, right PostingList, window uint32, ordered bool) PostingList {
	out := make([]Posting, 0, min(len(left.Items), len(right.Items)))
	i := 0
	j := 0
	for i < len(left.Items) && j < len(right.Items) {
		a := left.Items[i]
		b := right.Items[j]
		switch {
		case a.DocID == b.DocID:
			positions := matchingPositions(a.Positions, b.Positions, window, ordered)
			if len(positions) > 0 {
				out = append(out, Posting{DocID: a.DocID, TF: uint32(len(positions)), Positions: positions})
			}
			i++
			j++
		case a.DocID < b.DocID:
			i = advanceTo(left, i, b.DocID)
		default:
			j = advanceTo(right, j, a.DocID)
		}
	}
	return newPostingList(out)
}

func matchingPositions(left []uint32, right []uint32, window uint32, ordered bool) []uint32 {
	out := make([]uint32, 0)
	i := 0
	j := 0
	for i < len(left) && j < len(right) {
		a := left[i]
		b := right[j]
		if ordered {
			if b == a+1 {
				out = append(out, a)
				i++
				j++
				continue
			}
			if b <= a {
				j++
			} else {
				i++
			}
			continue
		}
		if near(a, b, window) {
			out = append(out, a)
			i++
			j++
			continue
		}
		if a < b {
			i++
		} else {
			j++
		}
	}
	return out
}

func near(a uint32, b uint32, window uint32) bool {
	if a > b {
		return a-b <= window
	}
	return b-a <= window
}

func mergePosting(left Posting, right Posting) Posting {
	if len(left.Positions) == 0 {
		return right
	}
	if len(right.Positions) == 0 {
		return left
	}
	positions := append(append([]uint32{}, left.Positions...), right.Positions...)
	sort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })
	positions = uniquePositions(positions)
	return Posting{DocID: left.DocID, TF: left.TF + right.TF, Positions: positions}
}

func uniquePositions(values []uint32) []uint32 {
	if len(values) < 2 {
		return values
	}
	write := 1
	for read := 1; read < len(values); read++ {
		if values[read] == values[write-1] {
			continue
		}
		values[write] = values[read]
		write++
	}
	return values[:write]
}

func advanceTo(list PostingList, pos int, target uint32) int {
	step := list.SkipStep
	if step < 1 {
		step = 1
	}
	for pos+step < len(list.Items) && list.Items[pos+step].DocID < target {
		pos += step
	}
	for pos < len(list.Items) && list.Items[pos].DocID < target {
		pos++
	}
	return pos
}
