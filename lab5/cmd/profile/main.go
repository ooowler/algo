package main

import (
	"fmt"
	"lab5/search"
	"os"
	"runtime"
	"runtime/pprof"
)

func main() {
	if err := os.MkdirAll("profiles", 0o755); err != nil {
		fmt.Println(err)
		return
	}
	cpu, err := os.Create("profiles/search_cpu.prof")
	if err != nil {
		fmt.Println(err)
		return
	}
	pprof.StartCPUProfile(cpu)
	idx := search.Build(search.SyntheticDocuments(20000))
	queries := []string{
		"alpha AND beta",
		"alpha OR gamma",
		"history AND NOT (russia AND china)",
		"quick ADJ brown",
		"russia NEAR/3 china",
	}
	for i := 0; i < 3000; i++ {
		_, _ = idx.Search(queries[i%len(queries)], 20)
	}
	pprof.StopCPUProfile()
	cpu.Close()

	runtime.GC()
	mem, err := os.Create("profiles/search_mem.prof")
	if err != nil {
		fmt.Println(err)
		return
	}
	pprof.WriteHeapProfile(mem)
	mem.Close()
	fmt.Println("profiles/search_cpu.prof")
	fmt.Println("profiles/search_mem.prof")
}
