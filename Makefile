.PHONY: build bench bench-compare bench-go bench-profile bench-tokens test clean

# ── build ──────────────────────────────────────────────────────────────────────

build:
	go build -o aict .

# ── tests ─────────────────────────────────────────────────────────────────────

test:
	go test ./... -timeout 60s

# ── external benchmarks (aict binary vs GNU tools) ────────────────────────────

bench: build
	go run ./cmd/bench -runs 7

bench-baseline: build
	go run ./cmd/bench -runs 7 -output benchmarks/baseline.xml
	@echo "Baseline written to benchmarks/baseline.xml"

bench-compare: build
	go run ./cmd/bench -runs 7 -compare benchmarks/baseline.xml

# ── token benchmarks (context-window cost vs GNU tools) ──────────────────────

bench-tokens: build
	go run ./cmd/tokenbench -aict ./aict -samples benchmarks/token-samples
	@echo "Real tokenizer counts: python benchmarks/count_tokens.py benchmarks/token-samples"

# ── Go testing.B benchmarks (internal, no subprocess) ────────────────────────

bench-go:
	go test -bench=. -benchmem -count=5 -timeout=180s \
		./tools/grep/ ./tools/cat/ ./tools/ls/

# Compare two runs with benchstat (install: go install golang.org/x/perf/cmd/benchstat@latest)
bench-stat:
	go test -bench=. -count=10 -timeout=300s ./tools/grep/ ./tools/cat/ ./tools/ls/ \
		| tee /tmp/bench-new.txt
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat /tmp/bench-new.txt; \
	else \
		echo "Install benchstat: go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

# Profile grep's regexp hot path
bench-profile:
	go test -bench=BenchmarkGrep_Regex_100k -cpuprofile=cpu.prof \
		-benchmem -count=1 -timeout=60s ./tools/grep/
	go tool pprof -http=:6060 cpu.prof

# ── clean ─────────────────────────────────────────────────────────────────────

clean:
	rm -f aict bench cpu.prof
