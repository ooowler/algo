package search

import (
	"reflect"
	"testing"
)

func TestEncodeUint32sRoundTrip(t *testing.T) {
	cases := [][]uint32{
		nil,
		{0, 0, 0, 0},
		{1, 2, 3, 127, 128, 1024, 1 << 20},
		makeSequence(513),
	}
	for _, values := range cases {
		encoded := EncodeUint32s(values)
		decoded, used, err := DecodeUint32s(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if used != len(encoded) {
			t.Fatalf("used=%d len=%d", used, len(encoded))
		}
		if !equalUint32s(values, decoded) {
			t.Fatalf("values mismatch\nwant=%v\ngot=%v", values, decoded)
		}
	}
}

func TestPostingCodecRoundTrip(t *testing.T) {
	list := newPostingList([]Posting{
		{DocID: 2, TF: 2, Positions: []uint32{1, 4}},
		{DocID: 9, TF: 3, Positions: []uint32{0, 7, 8}},
		{DocID: 20, TF: 1, Positions: []uint32{3}},
	})
	decoded, err := decodePostings(encodePostings(list))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(list.Items, decoded.Items) {
		t.Fatalf("posting mismatch\nwant=%v\ngot=%v", list.Items, decoded.Items)
	}
}

func makeSequence(n int) []uint32 {
	values := make([]uint32, n)
	for i := range values {
		values[i] = uint32((i * i) % 100000)
	}
	return values
}

func equalUint32s(a []uint32, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
