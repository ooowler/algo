package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"

	"geo/geosearch"
)

const (
	pointCount   = 200000
	queryCount   = 2048
	searchRadius = 500000.0
)

var sink int

func main() {
	if err := os.MkdirAll("profiles", 0755); err != nil {
		fmt.Println(err)
		return
	}

	cpuPath := "profiles/geosearch_cpu.prof"
	memPath := "profiles/geosearch_mem.prof"
	if err := writeCPU(cpuPath, prepare(pointCount)); err != nil {
		fmt.Println(err)
		return
	}
	if err := writeMem(memPath, prepare(pointCount)); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("geosearch -> %s, %s\n", cpuPath, memPath)
}

func prepare(n int) func() {
	pts := genPoints(n)
	queries := genPoints(queryCount)
	radii := []float64{10000, 100000, 500000, 2000000}

	return func() {
		kd := geosearch.Build(pts)
		grid := geosearch.NewGrid(searchRadius / 111320.0)
		naive := &geosearch.NaiveIndex{}
		for _, p := range pts {
			grid.Add(p)
			naive.Add(p)
		}

		total := 0
		for i := 0; i < len(queries)*8; i++ {
			center := queries[i&(len(queries)-1)]
			r := radii[i&3]
			total += len(kd.Search(center, r))
			total += len(grid.Search(center, r))
			if i&63 == 0 {
				total += len(naive.Search(center, r))
			}
		}
		sink += total
	}
}

func genPoints(n int) []geosearch.Point {
	rng := rand.New(rand.NewSource(42))
	pts := make([]geosearch.Point, n)
	for i := range pts {
		pts[i] = geosearch.Point{
			ID:  i,
			Lat: rng.Float64()*170 - 85,
			Lng: rng.Float64()*360 - 180,
		}
	}
	return pts
}

func writeCPU(path string, fn func()) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := pprof.StartCPUProfile(file); err != nil {
		return err
	}
	fn()
	pprof.StopCPUProfile()
	return nil
}

func writeMem(path string, fn func()) error {
	runtime.GC()
	fn()
	runtime.GC()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return pprof.WriteHeapProfile(file)
}
