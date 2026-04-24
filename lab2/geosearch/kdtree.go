package geosearch

import "math"

type kdNode struct {
	p           Point
	left, right *kdNode
}

type KDTree struct {
	root *kdNode
}

func Build(pts []Point) *KDTree {
	ps := make([]Point, len(pts))
	for i, p := range pts {
		ps[i] = normalizePoint(p)
	}
	return &KDTree{root: buildKD(ps, 0)}
}

func buildKD(pts []Point, depth int) *kdNode {
	if len(pts) == 0 {
		return nil
	}
	mid := len(pts) / 2
	if depth&1 == 0 {
		nthByLat(pts, mid)
	} else {
		nthByLng(pts, mid)
	}
	return &kdNode{
		p:     pts[mid],
		left:  buildKD(pts[:mid], depth+1),
		right: buildKD(pts[mid+1:], depth+1),
	}
}

func nthByLat(s []Point, k int) {
	for len(s) > 1 {
		p := s[len(s)>>1].Lat
		lo, hi, i := 0, len(s)-1, 0
		for i <= hi {
			if s[i].Lat < p {
				s[lo], s[i] = s[i], s[lo]
				lo++
				i++
			} else if s[i].Lat > p {
				s[i], s[hi] = s[hi], s[i]
				hi--
			} else {
				i++
			}
		}
		if k < lo {
			s = s[:lo]
		} else if k > hi {
			s = s[hi+1:]
			k -= hi + 1
		} else {
			return
		}
	}
}

func nthByLng(s []Point, k int) {
	for len(s) > 1 {
		p := s[len(s)>>1].Lng
		lo, hi, i := 0, len(s)-1, 0
		for i <= hi {
			if s[i].Lng < p {
				s[lo], s[i] = s[i], s[lo]
				lo++
				i++
			} else if s[i].Lng > p {
				s[i], s[hi] = s[hi], s[i]
				hi--
			} else {
				i++
			}
		}
		if k < lo {
			s = s[:lo]
		} else if k > hi {
			s = s[hi+1:]
			k -= hi + 1
		} else {
			return
		}
	}
}

func (t *KDTree) Add(p Point) {
	t.root = insertKD(t.root, normalizePoint(p), 0)
}

func insertKD(nd *kdNode, p Point, depth int) *kdNode {
	if nd == nil {
		return &kdNode{p: p}
	}
	if depth&1 == 0 {
		if p.Lat < nd.p.Lat {
			nd.left = insertKD(nd.left, p, depth+1)
		} else {
			nd.right = insertKD(nd.right, p, depth+1)
		}
	} else {
		if p.Lng < nd.p.Lng {
			nd.left = insertKD(nd.left, p, depth+1)
		} else {
			nd.right = insertKD(nd.right, p, depth+1)
		}
	}
	return nd
}

func (t *KDTree) Search(center Point, radiusM float64) []Point {
	center = normalizePoint(center)
	latDeg := radiusM / 111320.0
	cosCLat := math.Cos(center.Lat * rad)
	lngDeg := lngRadiusDeg(center.Lat, radiusM)
	maxH := radiusH(radiusM)
	var out []Point
	searchKD(t.root, center, maxH, latDeg, lngDeg, cosCLat, lngWindowWraps(center.Lng, lngDeg), 0, &out)
	return out
}

func searchKD(nd *kdNode, c Point, maxH, latDeg, lngDeg, cosCLat float64, wrapLng bool, depth int, out *[]Point) {
	if nd == nil {
		return
	}
	if haversineH(nd.p, c, cosCLat) <= maxH {
		*out = append(*out, nd.p)
	}
	var diff, bound float64
	if depth&1 == 0 {
		diff = c.Lat - nd.p.Lat
		bound = latDeg
	} else {
		diff = c.Lng - nd.p.Lng
		bound = lngDeg
	}
	var near, far *kdNode
	if diff <= 0 {
		near, far = nd.left, nd.right
	} else {
		near, far = nd.right, nd.left
	}
	searchKD(near, c, maxH, latDeg, lngDeg, cosCLat, wrapLng, depth+1, out)
	if depth&1 == 1 && wrapLng {
		searchKD(far, c, maxH, latDeg, lngDeg, cosCLat, wrapLng, depth+1, out)
		return
	}
	if math.Abs(diff) <= bound {
		searchKD(far, c, maxH, latDeg, lngDeg, cosCLat, wrapLng, depth+1, out)
	}
}
