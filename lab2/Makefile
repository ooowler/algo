.PHONY: test bench profile charts clean

GOCACHE_DIR := $(PWD)/.gocache
GOTMPDIR_DIR := $(PWD)/.tmp

test:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR)
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go test ./... -timeout 120s -count=1

bench:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR)
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go test ./geosearch -run=^$$ -bench=. -benchmem -benchtime=500ms -count=1

profile:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) results figures
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go run ./cmd/profile
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go tool pprof -top -nodecount=10 profiles/geosearch_cpu.prof > results/profile_cpu.txt
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) go tool pprof -top -sample_index=alloc_space -nodecount=10 profiles/geosearch_mem.prof > results/profile_mem.txt
	env GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) TMPDIR=$(GOTMPDIR_DIR) PPROF_PORT=17352 bash ../lab1/scripts/export_pprof_flame_html.sh profiles/geosearch_cpu.prof figures/pprof_geosearch_flamegraph.html

charts:
	@PY=$$(test -x ../.venv/bin/python && echo ../.venv/bin/python || echo python3); \
	 $$PY scripts/build_charts_lab2.py

clean:
	rm -rf .gocache .tmp figures/ results/ profiles/
