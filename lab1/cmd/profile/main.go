package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"

	"algo/hashtable"
	"algo/lsh"
	"algo/perfecthash"
)

type workload struct {
	name    string
	n       int
	prepare func(int) preparedWorkload
}

type preparedWorkload struct {
	run     func()
	cleanup func()
}

var workloads = []workload{
	{name: "hashtable", n: 65536, prepare: prepareHashtable},
	{name: "perfecthash", n: 262144, prepare: preparePerfectHash},
	{name: "lsh", n: 32000, prepare: prepareLSH},
}

func main() {
	only := flag.String("only", "", "hashtable|perfecthash|lsh")
	flag.Parse()

	if err := os.MkdirAll("profiles", 0755); err != nil {
		fmt.Println(err)
		return
	}

	for _, workload := range workloads {
		if *only != "" && workload.name != *only {
			continue
		}
		cpuPath := fmt.Sprintf("profiles/%s_cpu.prof", workload.name)
		memPath := fmt.Sprintf("profiles/%s_mem.prof", workload.name)
		if err := writeCPU(cpuPath, workload.prepare(workload.n)); err != nil {
			fmt.Println(err)
			return
		}
		if err := writeMem(memPath, workload.prepare(workload.n)); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%s (n=%d) -> %s, %s\n", workload.name, workload.n, cpuPath, memPath)
	}
}

func prepareHashtable(n int) preparedWorkload {
	keys := makeProfileKeys("k", n)
	values := makeProfileValues("value", n)
	ops := n / 2
	extraKeys := makeProfileKeys("x", ops)
	extraValues := makeProfileValues("extra", ops)
	getKeys := make([]string, ops)
	updateValues := make([]string, ops)
	for i := 0; i < ops; i++ {
		getKeys[i] = keys[(i*17)%n]
		updateValues[i] = values[(i*7)%n]
	}

	var dir string
	var table *hashtable.DiskHashTable
	return preparedWorkload{
		run: func() {
			var err error
			dir, err = os.MkdirTemp("", "dht-prof-*")
			if err != nil {
				panic(err)
			}
			table, err = hashtable.New(dir, max(64, n/16))
			if err != nil {
				panic(err)
			}

			for i := range keys {
				must(table.Set(keys[i], values[i]))
			}
			must(table.Flush())
			for pass := 0; pass < 2; pass++ {
				for i := 0; i < ops; i++ {
					must(table.Set(extraKeys[i], extraValues[i]))
					must(table.Set(keys[i%n], updateValues[i]))
					_, _, err = table.Get(getKeys[i])
					must(err)
					_, _, err = table.Get(extraKeys[i])
					must(err)
					if i > 0 {
						must(table.Delete(extraKeys[i-1]))
					}
				}
				must(table.Flush())
			}
		},
		cleanup: func() {
			if table != nil {
				must(table.Close())
			}
			if dir != "" {
				_ = os.RemoveAll(dir)
			}
			keys = nil
			values = nil
			extraKeys = nil
			extraValues = nil
			getKeys = nil
			updateValues = nil
		},
	}
}

func preparePerfectHash(n int) preparedWorkload {
	keys := make([]string, n)
	values := make([]string, n)
	keyWidth := len(strconv.Itoa(262144 - 1))
	valueWidth := len(strconv.Itoa(262144 * 2))
	for i := range keys {
		keys[i] = fmt.Sprintf("k%0*d", keyWidth, i)
		values[i] = fmt.Sprintf("v%0*d", valueWidth, i*2)
	}
	reads := n * 12
	lookupKeys := make([]string, reads)
	for i := range lookupKeys {
		lookupKeys[i] = keys[i%n]
	}

	return preparedWorkload{
		run: func() {
			for pass := 0; pass < 12; pass++ {
				index := perfecthash.Build(keys, values)
				for _, key := range lookupKeys {
					index.Get(key)
				}
			}
		},
		cleanup: func() {
			keys = nil
			values = nil
			lookupKeys = nil
		},
	}
}

func prepareLSH(n int) preparedWorkload {
	const threshold = 1.5
	baseSets := make([][]lsh.Point3D, 6)
	addSets := make([][]lsh.Point3D, 6)
	for pass := range baseSets {
		baseSets[pass] = lsh.GenerateDataset(n, threshold, int64(42+pass))
		addSets[pass] = lsh.GenerateDataset(max(1024, n/6), threshold, int64(99+pass))
	}

	return preparedWorkload{
		run: func() {
			for pass := range baseSets {
				index := lsh.BuildIndex(12, 10, threshold, rand.New(rand.NewSource(int64(7+pass))), baseSets[pass])
				for _, point := range addSets[pass] {
					index.Add(point)
				}
				for i := 0; i < 3; i++ {
					index.FindDuplicates()
				}
			}
		},
		cleanup: func() {
			baseSets = nil
			addSets = nil
		},
	}
}

func makeProfileKeys(prefix string, count int) []string {
	keys := make([]string, count)
	for i := range keys {
		keys[i] = fmt.Sprintf("%s%07d", prefix, i)
	}
	return keys
}

func makeProfileValues(prefix string, count int) []string {
	values := make([]string, count)
	for i := range values {
		values[i] = fmt.Sprintf("%s-%07d", prefix, i)
	}
	return values
}

func writeCPU(path string, work preparedWorkload) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := pprof.StartCPUProfile(file); err != nil {
		return err
	}
	work.run()
	pprof.StopCPUProfile()
	work.cleanup()
	return nil
}

func writeMem(path string, work preparedWorkload) error {
	runtime.GC()
	work.run()
	work.cleanup()
	runtime.GC()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return pprof.WriteHeapProfile(file)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
