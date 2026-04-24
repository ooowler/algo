.PHONY: test bench charts clean

test:
	go test ./... -timeout 120s -count=1

bench:
	go test ./geosearch -run=^$$ -bench=. -benchmem -benchtime=500ms -count=1

charts:
	@PY=$$(test -x ../.venv/bin/python && echo ../.venv/bin/python || echo python3); \
	 $$PY scripts/build_charts_lab2.py

clean:
	rm -rf figures/ results/
