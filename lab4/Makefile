.PHONY: test race bench profile charts report clean

GOCACHE_DIR := $(PWD)/.gocache
GOTMPDIR_DIR := $(PWD)/.tmp
BENCH_COUNT ?= 5
BENCHTIME ?= 200ms

test:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR)
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go test ./... -timeout 180s -count=1

race:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR)
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go test -race ./concurrentmap -timeout 300s -count=1

bench:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) results
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go test ./concurrentmap -run=^$$ -bench='Growth|Parallel' -benchmem -benchtime=$(BENCHTIME) -count=$(BENCH_COUNT) -timeout=1800s > results/bench.txt

profile:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) results
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go run ./cmd/profile
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go tool pprof -top -nodecount=10 profiles/concurrent_cpu.prof > results/profile_cpu.txt
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go tool pprof -top -sample_index=alloc_space -nodecount=10 profiles/concurrent_mem.prof > results/profile_mem.txt

charts:
	mkdir -p .mplconfig .cache
	@PY=$$(test -x ../.venv/bin/python && echo ../.venv/bin/python || echo python3); \
	 MPLCONFIGDIR=$(PWD)/.mplconfig XDG_CACHE_HOME=$(PWD)/.cache $$PY scripts/build_charts.py

report: bench profile charts

clean:
	rm -rf .gocache .tmp results figures profiles
