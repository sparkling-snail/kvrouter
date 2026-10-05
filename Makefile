.PHONY: build test bench seeds drift summary clean

build:
	go build -o bin/ ./cmd/...

test:
	go vet ./...
	go test -race ./...

bench: build
	./scripts/bench.sh

# Regenerates results/seed{1,2,3}/ (~6 min), the index-drift runs, then the README tables + chart.
seeds: build
	for s in 1 2 3; do SEED=$$s OUT=results/seed$$s ./scripts/bench.sh; done

drift: build
	./scripts/index_drift.sh

summary: .venv
	.venv/bin/python scripts/summarize.py

.venv: requirements.txt
	python3 -m venv .venv && .venv/bin/pip install -q -r requirements.txt && touch .venv

clean:
	rm -rf bin logs results/*.json results/*.prom results/RESULTS.md
