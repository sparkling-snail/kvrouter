.PHONY: build test bench clean

build:
	go build -o bin/ ./cmd/...

test:
	go vet ./...
	go test -race ./...

bench: build
	./scripts/bench.sh

clean:
	rm -rf bin logs results/*.json results/*.prom results/RESULTS.md
