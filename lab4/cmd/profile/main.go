package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"lab4/concurrentmap"
)

func main() {
	if err := os.MkdirAll("profiles", 0755); err != nil {
		fmt.Println(err)
		return
	}

	cpuPath := "profiles/concurrent_cpu.prof"
	memPath := "profiles/concurrent_mem.prof"
	if err := writeCPU(cpuPath, prepare(262144)); err != nil {
		fmt.Println(err)
		return
	}
	if err := writeMem(memPath, prepare(262144)); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("concurrentmap -> %s, %s\n", cpuPath, memPath)
}

func prepare(n int) func() {
	keys := makeKeys(0, n)
	hot := makeProbeKeys(keys, 4096)
	extra := makeKeys(n, max(4096, n/8))

	return func() {
		m := concurrentmap.NewStringMap[int](bucketCountFor(n))
		for i := range keys {
			m.Put(keys[i], i)
		}

		var wg sync.WaitGroup
		workers := 4
		mask := len(hot) - 1
		opsPerWorker := len(hot) * 256
		for worker := 0; worker < workers; worker++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < opsPerWorker; i++ {
					key := hot[(i+id*17)&mask]
					switch {
					case i&31 == 0:
						m.Merge(key, 1, func(a, b int) int { return a + b })
					case i&63 == 0:
						m.Put(extra[(i+id)%len(extra)], i)
					default:
						_, _ = m.Get(key)
					}
				}
			}(worker)
		}
		wg.Wait()
	}
}

func makeKeys(start, count int) []string {
	keys := make([]string, count)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%07d", start+i)
	}
	return keys
}

func makeProbeKeys(keys []string, maxProbes int) []string {
	if len(keys) == 0 {
		return nil
	}
	count := len(keys)
	if count > maxProbes {
		count = maxProbes
	}
	count = nextPow2(count)
	out := make([]string, count)
	for i := range out {
		out[i] = keys[int((uint64(i)*11400714819323198485)%uint64(len(keys)))]
	}
	return out
}

func bucketCountFor(n int) int {
	count := n / 8
	if count < 64 {
		count = 64
	}
	return nextPow2(count)
}

func nextPow2(v int) int {
	if v <= 1 {
		return 1
	}
	n := 1
	for n < v {
		n <<= 1
	}
	return n
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
