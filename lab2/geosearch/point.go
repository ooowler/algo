package geosearch

import "math"

const earthRadius = 6371000.0
const rad = math.Pi / 180
const minLat = -90.0
const maxLat = 90.0

var maxLngEdge = math.Nextafter(180.0, -1.0)

type Point struct {
	ID  int
	Lat float64
	Lng float64
}

func radiusH(radiusM float64) float64 {
	t := math.Sin(radiusM / (2 * earthRadius))
	return t * t
}

func normalizeLng(lng float64) float64 {
	for lng < -180 {
		lng += 360
	}
	for lng >= 180 {
		lng -= 360
	}
	return lng
}

func clampLat(lat float64) float64 {
	if lat < minLat {
		return minLat
	}
	if lat > maxLat {
		return maxLat
	}
	return lat
}

func normalizePoint(p Point) Point {
	p.Lat = clampLat(p.Lat)
	p.Lng = normalizeLng(p.Lng)
	return p
}

func lngDelta(a, b float64) float64 {
	return normalizeLng(a - b)
}

func lngWindowWraps(centerLng, lngDeg float64) bool {
	centerLng = normalizeLng(centerLng)
	return centerLng-lngDeg < -180 || centerLng+lngDeg >= 180
}

func lngRadiusDeg(centerLat, radiusM float64) float64 {
	latDeg := radiusM / 111320.0
	low := clampLat(centerLat - latDeg)
	high := clampLat(centerLat + latDeg)
	edge := math.Max(math.Abs(low), math.Abs(high))
	cosLat := math.Cos(edge * rad)
	if cosLat < 0.001 {
		return 180
	}
	lngDeg := radiusM / (111320.0 * cosLat)
	if lngDeg > 180 {
		return 180
	}
	return lngDeg
}

func haversineH(a, b Point, cosBLat float64) float64 {
	la1 := a.Lat * rad
	s1 := math.Sin((b.Lat-a.Lat)*rad / 2)
	s2 := math.Sin(lngDelta(b.Lng, a.Lng) * rad / 2)
	return s1*s1 + math.Cos(la1)*cosBLat*s2*s2
}

func DistanceM(a, b Point) float64 {
	a = normalizePoint(a)
	b = normalizePoint(b)
	return earthRadius * 2 * math.Asin(math.Sqrt(haversineH(a, b, math.Cos(b.Lat*rad))))
}

type Index interface {
	Add(p Point)
	Search(center Point, radiusM float64) []Point
}

type NaiveIndex struct {
	pts []Point
}

func (n *NaiveIndex) Add(p Point) {
	n.pts = append(n.pts, normalizePoint(p))
}

func (n *NaiveIndex) Search(center Point, radiusM float64) []Point {
	center = normalizePoint(center)
	cosCLat := math.Cos(center.Lat * rad)
	maxH := radiusH(radiusM)
	var out []Point
	for _, p := range n.pts {
		if haversineH(p, center, cosCLat) <= maxH {
			out = append(out, p)
		}
	}
	return out
}
