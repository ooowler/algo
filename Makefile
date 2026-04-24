.PHONY: test bench bench-scale bench-growth profile trace figures figures-pprof figures-flame-html bench-disk-cloud figures-bar charts-mpl lab1-metrics charts-mpl-full lab1-full-report lab2 lab2-bench lab2-charts lab3 lab3-demo lab4 lab4-test lab4-race lab4-report clean all

LAB1 = lab1
LAB2 = lab2
LAB4 = lab4

test:
	go test -C $(LAB1) ./... -timeout 300s -count=1

bench:
	go test -C $(LAB1) ./... -run=^$$ -bench=. -benchmem -benchtime=2s -count=1

bench-scale:
	go test -C $(LAB1) ./hashtable ./perfecthash ./lsh -run=^$$ -bench='Small|Medium|Large|_1K|_10K|_50K' -benchmem -benchtime=2s -count=1

bench-growth:
	mkdir -p $(LAB1)/results
	GOCACHE=$(PWD)/$(LAB1)/.gocache go test -C $(LAB1) ./hashtable ./perfecthash ./lsh -run=^$$ -bench=BenchmarkGrowth -benchmem -benchtime=200ms -count=5 -timeout=1800s | tee $(LAB1)/results/growth_bench.txt

lab1-full-report:
	chmod +x $(LAB1)/scripts/full_report_run.sh && cd $(LAB1) && bash scripts/full_report_run.sh

profile:
	go run -C $(LAB1) ./cmd/profile

trace-lsh:
	mkdir -p $(LAB1)/profiles
	go test -C $(LAB1) ./lsh -run=^$$ -bench=BenchmarkFindDuplicates_Medium -benchtime=300ms -trace=$(LAB1)/profiles/lsh_trace_medium.out
	@echo "go tool trace $(LAB1)/profiles/lsh_trace_medium.out"

trace-hashtable:
	mkdir -p $(LAB1)/profiles
	go test -C $(LAB1) ./hashtable -run=^$$ -bench=BenchmarkSet_Medium -benchtime=300ms -trace=$(LAB1)/profiles/hashtable_trace_set.out
	@echo "go tool trace $(LAB1)/profiles/hashtable_trace_set.out"

figures:
	$(MAKE) charts-mpl

lab1-metrics:
	GOCACHE=$(PWD)/$(LAB1)/.gocache PHF_METRICS=1 go test -C $(LAB1) ./perfecthash -run=TestPHF_Metrics -count=1 -timeout=300s
	GOCACHE=$(PWD)/$(LAB1)/.gocache LSH_METRICS=1 go test -C $(LAB1) ./lsh -run=TestLSH_Metrics -count=1 -timeout=300s

charts-mpl:
	@if [ -x $$HOME/venv/bin/python ]; then $$HOME/venv/bin/python $(LAB1)/scripts/build_charts_matplotlib.py; elif [ -x .venv/bin/python ]; then .venv/bin/python $(LAB1)/scripts/build_charts_matplotlib.py; else python3 $(LAB1)/scripts/build_charts_matplotlib.py; fi

charts-mpl-full: charts-mpl

figures-pprof:
	$(MAKE) charts-mpl

bench-disk-cloud:
	$(MAKE) charts-mpl

figures-flame-html:
	chmod +x $(LAB1)/scripts/export_pprof_flame_html.sh && $(LAB1)/scripts/export_pprof_flame_html.sh

figures-bar:
	$(MAKE) charts-mpl

lab3:
	@mkdir -p lab3/.mplconfig lab3/.cache
	@if [ -x lab3/.venv312/bin/python ]; then MPLCONFIGDIR=$(PWD)/lab3/.mplconfig XDG_CACHE_HOME=$(PWD)/lab3/.cache lab3/.venv312/bin/python lab3/run.py; elif [ -x lab3/.venv/bin/python ]; then MPLCONFIGDIR=$(PWD)/lab3/.mplconfig XDG_CACHE_HOME=$(PWD)/lab3/.cache lab3/.venv/bin/python lab3/run.py; elif [ -f $$HOME/venv/bin/python ]; then MPLCONFIGDIR=$(PWD)/lab3/.mplconfig XDG_CACHE_HOME=$(PWD)/lab3/.cache $$HOME/venv/bin/python lab3/run.py; elif command -v python3.12 >/dev/null 2>&1; then MPLCONFIGDIR=$(PWD)/lab3/.mplconfig XDG_CACHE_HOME=$(PWD)/lab3/.cache python3.12 lab3/run.py; else MPLCONFIGDIR=$(PWD)/lab3/.mplconfig XDG_CACHE_HOME=$(PWD)/lab3/.cache python3 lab3/run.py; fi

lab3-demo:
	@LAB3_DEMO=1 $(MAKE) lab3

lab2:
	$(MAKE) -C $(LAB2) test

lab2-bench:
	$(MAKE) -C $(LAB2) bench

lab2-charts:
	$(MAKE) -C $(LAB2) charts

lab4:
	$(MAKE) -C $(LAB4) report

lab4-test:
	$(MAKE) -C $(LAB4) test

lab4-race:
	$(MAKE) -C $(LAB4) race

lab4-report:
	$(MAKE) -C $(LAB4) report

clean:
	rm -f $(LAB1)/profiles/*.prof $(LAB1)/profiles/*.out $(LAB1)/results/*.txt $(LAB1)/results/*.md $(LAB1)/results/*.json

all: test figures
