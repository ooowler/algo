package geosearch

import "math"

type cellKey [2]int32
type cellRange struct {
	min int32
	max int32
}

type GridIndex struct {
	cellDeg float64
	cells   map[cellKey][]Point
}

func NewGrid(cellDeg float64) *GridIndex {
	return &GridIndex{cellDeg: cellDeg, cells: make(map[cellKey][]Point)}
}

func (g *GridIndex) cellOf(lat, lng float64) cellKey {
	return cellKey{
		int32(math.Floor(clampLat(lat) / g.cellDeg)),
		int32(math.Floor(normalizeLng(lng) / g.cellDeg)),
	}
}

func (g *GridIndex) lngRanges(centerLng, lngDeg float64) []cellRange {
	if lngDeg >= 180 {
		return []cellRange{{
			min: int32(math.Floor(-180 / g.cellDeg)),
			max: int32(math.Floor(maxLngEdge / g.cellDeg)),
		}}
	}
	minLng := normalizeLng(centerLng - lngDeg)
	maxLngBound := normalizeLng(centerLng + lngDeg)
	if minLng <= maxLngBound {
		return []cellRange{{
			min: int32(math.Floor(minLng / g.cellDeg)),
			max: int32(math.Floor(maxLngBound / g.cellDeg)),
		}}
	}
	return []cellRange{
		{
			min: int32(math.Floor(-180 / g.cellDeg)),
			max: int32(math.Floor(maxLngBound / g.cellDeg)),
		},
		{
			min: int32(math.Floor(minLng / g.cellDeg)),
			max: int32(math.Floor(maxLngEdge / g.cellDeg)),
		},
	}
}

func (g *GridIndex) Add(p Point) {
	p = normalizePoint(p)
	k := g.cellOf(p.Lat, p.Lng)
	g.cells[k] = append(g.cells[k], p)
}

func (g *GridIndex) Search(center Point, radiusM float64) []Point {
	center = normalizePoint(center)
	latDeg := radiusM / 111320.0
	cosCLat := math.Cos(center.Lat * rad)
	lngDeg := lngRadiusDeg(center.Lat, radiusM)
	minLat := int32(math.Floor(clampLat(center.Lat-latDeg) / g.cellDeg))
	maxLat := int32(math.Floor(clampLat(center.Lat+latDeg) / g.cellDeg))
	maxH := radiusH(radiusM)
	var out []Point
	lngRanges := g.lngRanges(center.Lng, lngDeg)
	for la := minLat; la <= maxLat; la++ {
		for _, span := range lngRanges {
			for lo := span.min; lo <= span.max; lo++ {
				for _, p := range g.cells[cellKey{la, lo}] {
					if haversineH(p, center, cosCLat) <= maxH {
						out = append(out, p)
					}
				}
			}
		}
	}
	return out
}
