.PHONY: test bench profile demo server clean

GOCACHE_DIR := $(PWD)/.gocache
GOTMPDIR_DIR := $(PWD)/.tmp
BENCHTIME ?= 200ms
BENCH_COUNT ?= 3

test:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR)
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) TMPDIR=$(GOTMPDIR_DIR) go test ./... -timeout 180s -count=1

bench:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) results
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) TMPDIR=$(GOTMPDIR_DIR) go test ./search -run=^$$ -bench='Queries|Compression' -benchmem -benchtime=$(BENCHTIME) -count=$(BENCH_COUNT) -timeout=1800s > results/bench.txt

profile:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) results profiles
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) TMPDIR=$(GOTMPDIR_DIR) go run ./cmd/profile
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go tool pprof -top -nodecount=15 profiles/search_cpu.prof > results/profile_cpu.txt
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go tool pprof -top -sample_index=alloc_space -nodecount=15 profiles/search_mem.prof > results/profile_mem.txt

demo:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) data
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) TMPDIR=$(GOTMPDIR_DIR) go run ./cmd/lab5 demo -index data/demo.lab5

server: demo
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) TMPDIR=$(GOTMPDIR_DIR) go run ./cmd/server -index data/demo.lab5

clean:
	rm -rf .gocache .tmp data/*.lab5 results profiles
