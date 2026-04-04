package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"strconv"

	"algo/hashtable"
	"algo/lsh"
	"algo/perfecthash"
)

type tier struct {
	name    string
	nDisk   int
	nPH     int
	nLSH    int
}

var TIERS = []tier{
	{name: "small", nDisk: 4000, nPH: 4000, nLSH: 800},
	{name: "medium", nDisk: 25000, nPH: 25000, nLSH: 5000},
	{name: "large", nDisk: 80000, nPH: 60000, nLSH: 12000},
}

type workload struct {
	name string
	n    func(tier) int
	run  func(int)
}

var WORKLOADS = []workload{
	{
		name: "hashtable",
		n:    func(z tier) int { return z.nDisk },
		run:  runHashtable,
	},
	{
		name: "perfecthash",
		n:    func(z tier) int { return z.nPH },
		run:  runPerfectHash,
	},
	{
		name: "lsh",
		n:    func(z tier) int { return z.nLSH },
		run:  runLSH,
	},
}

func runHashtable(n int) {
	dir, _ := os.MkdirTemp("", "dht-prof-*")
	defer os.RemoveAll(dir)
	h, _ := hashtable.New(dir, 256)
	for i := 0; i < n; i++ {
		h.Set("key-"+strconv.Itoa(i), "value-"+strconv.Itoa(i))
	}
	for i := 0; i < n; i++ {
		h.Get("key-" + strconv.Itoa(i))
	}
	for i := 0; i < n/5; i++ {
		h.Delete("key-" + strconv.Itoa(i))
	}
}

func runPerfectHash(n int) {
	keys := make([]string, n)
	vals := make([]string, n)
	for i := range keys {
		keys[i] = "key-" + strconv.Itoa(i)
		vals[i] = strconv.Itoa(i)
	}
	idx := perfecthash.Build(keys, vals)
	for i := 0; i < n; i++ {
		idx.Get(keys[i])
	}
}

func runLSH(n int) {
	rng := rand.New(rand.NewSource(42))
	t := lsh.New(10, 8, 2.0, rng)
	for i := 0; i < n; i++ {
		t.Add(lsh.Point3D{
			X:  rng.Float64() * 100,
			Y:  rng.Float64() * 100,
			Z:  rng.Float64() * 100,
			ID: i,
		})
	}
	t.FindDuplicates()
}

func writeCPU(path string, n int, fn func(int)) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	pprof.StartCPUProfile(f)
	fn(n)
	pprof.StopCPUProfile()
	return nil
}

func writeMem(path string, n int, fn func(int)) error {
	fn(n)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pprof.WriteHeapProfile(f)
}

func main() {
	only := flag.String("only", "", "hashtable|perfecthash|lsh (empty = all)")
	flag.Parse()
	if err := os.MkdirAll("profiles", 0755); err != nil {
		fmt.Println(err)
		return
	}

	for _, w := range WORKLOADS {
		if *only != "" && w.name != *only {
			continue
		}
		for _, z := range TIERS {
			n := w.n(z)
			base := fmt.Sprintf("profiles/%s_%s", w.name, z.name)
			if err := writeCPU(base+"_cpu.prof", n, w.run); err != nil {
				fmt.Println(err)
				return
			}
			if err := writeMem(base+"_mem.prof", n, w.run); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("%s %s (n=%d) -> %s_{cpu,mem}.prof\n", w.name, z.name, n, base)
		}
	}
}
