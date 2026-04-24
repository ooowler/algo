package lsh

import (
	"math"
	"math/rand"
)

const duplicateFraction = 0.12

func GenerateDataset(n int, threshold float64, seed int64) []Point3D {
	if n <= 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(seed))
	dupCount := int(math.Round(float64(n) * duplicateFraction))
	if dupCount >= n {
		dupCount = n / 2
	}
	baseCount := n - dupCount
	if baseCount <= 0 {
		baseCount = n
		dupCount = 0
	}

	spacing := threshold * 4.5
	side := int(math.Ceil(math.Cbrt(float64(baseCount))))
	points := make([]Point3D, 0, n)
	for i := 0; i < baseCount; i++ {
		x := float64(i%side) * spacing
		y := float64((i/side)%side) * spacing
		z := float64(i/(side*side)) * spacing
		jitter := threshold * 0.08
		x += (rng.Float64()*2 - 1) * jitter
		y += (rng.Float64()*2 - 1) * jitter
		z += (rng.Float64()*2 - 1) * jitter
		points = append(points, Point3D{X: x, Y: y, Z: z, ID: len(points)})
	}

	if dupCount == 0 {
		return points
	}
	perm := rng.Perm(baseCount)
	for i := 0; i < dupCount; i++ {
		base := points[perm[i]]
		radius := threshold * (0.18 + 0.12*rng.Float64())
		phi := rng.Float64() * 2 * math.Pi
		costheta := rng.Float64()*2 - 1
		sintheta := math.Sqrt(1 - costheta*costheta)
		dx := radius * sintheta * math.Cos(phi)
		dy := radius * sintheta * math.Sin(phi)
		dz := radius * costheta
		points = append(points, Point3D{
			X:  base.X + dx,
			Y:  base.Y + dy,
			Z:  base.Z + dz,
			ID: len(points),
		})
	}
	return points
}

func BuildIndex(numTables int, extra int, threshold float64, rng *rand.Rand, points []Point3D) *LSH {
	expectedPoints := len(points)
	if extra > 0 {
		expectedPoints += extra
	}
	index := New(numTables, expectedPoints, threshold, rng)
	for _, point := range points {
		index.Add(point)
	}
	return index
}
