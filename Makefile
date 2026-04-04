.PHONY: test bench bench-scale profile trace figures figures-bar clean all

test:
	go test ./... -timeout 300s -count=1

bench:
	go test ./... -run=^$$ -bench=. -benchmem -benchtime=2s -count=1

bench-scale:
	go test ./hashtable ./perfecthash ./lsh -run=^$$ -bench='Small|Medium|Large|_1K|_10K|_50K' -benchmem -benchtime=2s -count=1

bench-growth:
	mkdir -p results
	go test ./hashtable ./perfecthash ./lsh -run=^$$ -bench=BenchmarkGrowth -benchmem -benchtime=150ms -count=1 -timeout=180s | tee results/growth_bench.txt

profile:
	go run ./cmd/profile

trace-lsh:
	mkdir -p profiles
	go test ./lsh -run=^$$ -bench=BenchmarkFindDuplicates_Medium -benchtime=300ms -trace=profiles/lsh_trace_medium.out
	@echo "go tool trace profiles/lsh_trace_medium.out"

trace-hashtable:
	mkdir -p profiles
	go test ./hashtable -run=^$$ -bench=BenchmarkSet_Medium -benchtime=300ms -trace=profiles/hashtable_trace_set.out
	@echo "go tool trace profiles/hashtable_trace_set.out"

figures:
	go run ./cmd/genfigures

charts-mpl:
	@if [ -x .venv/bin/python ]; then .venv/bin/python scripts/build_charts_matplotlib.py; else python3 scripts/build_charts_matplotlib.py; fi

figures-bar:
	python3 scripts/plot_benchmarks.py

clean:
	rm -f profiles/*.prof profiles/*.out results/*.txt

all: test figures
