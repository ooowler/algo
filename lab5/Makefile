.PHONY: corpus build stress profile flamegraph report server clean

GOCACHE_DIR := $(PWD)/.gocache
GOTMPDIR_DIR := $(PWD)/.tmp
GO_ENV := GOCACHE=$(GOCACHE_DIR) GOTMPDIR=$(GOTMPDIR_DIR) TMPDIR=$(GOTMPDIR_DIR)
SRC_UTF8 := $(wildcard data/src_utf8/*.html)

corpus:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) data
	env $(GO_ENV) go run ./cmd/lab5 import-libru -out data/russian_classics.jsonl $(SRC_UTF8)

build:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) data
	env $(GO_ENV) go run ./cmd/lab5 build -docs data/russian_classics.jsonl -index data/russian_classics.lab5

stress:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR) results profiles
	env $(GO_ENV) go run ./cmd/stress -index data/russian_classics.lab5 -corpus data/russian_classics.jsonl -sources data/src_utf8 -out results/stress.md -rounds 1000 -limit 10 -cpuprofile profiles/search_cpu.prof

profile:
	mkdir -p $(GOTMPDIR_DIR) results
	env TMPDIR=$(GOTMPDIR_DIR) go tool pprof -top -nodecount=15 profiles/search_cpu.prof > results/profile_cpu.txt

flamegraph:
	mkdir -p $(GOTMPDIR_DIR) figures
	env TMPDIR=$(GOTMPDIR_DIR) bash ../lab1/scripts/export_pprof_flame_html.sh profiles/search_cpu.prof figures/pprof_search_flamegraph.html

report: corpus build stress profile flamegraph

server:
	mkdir -p $(GOCACHE_DIR) $(GOTMPDIR_DIR)
	env $(GO_ENV) go run ./cmd/server -index data/russian_classics.lab5

clean:
	rm -rf .gocache .tmp
