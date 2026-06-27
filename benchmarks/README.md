# Performance Benchmarks

This directory contains the benchmark suite for aict.

## Quick start

```bash
# Build and run the full suite (7 runs, median reported)
make bench

# Compare against a committed baseline
make bench-compare

# Update the baseline after a performance improvement
make bench-baseline

# Go testing.B benchmarks (grep / cat / ls internal hot paths)
make bench-go

# Profile grep's regexp hot path with pprof
make bench-profile
```

Or without `make`:

```bash
go build -o aict .
go run ./cmd/bench -runs 7
go run ./cmd/bench -runs 7 -compare benchmarks/baseline.xml
go test -bench=. -benchmem -count=5 ./tools/grep/ ./tools/cat/ ./tools/ls/
```

## bench CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `-runs N` | 7 | Runs per suite (median reported) |
| `-output FILE` | stdout | Write XML results to file |
| `-compare FILE` | — | Compare against a baseline XML; emit `<delta>` per suite |
| `-threshold F` | 10.0 | Pass/fail ratio vs GNU baseline |
| `-aict PATH` | `./aict` | Path to aict binary |
| `-suites LIST` | all | Comma-separated suite names to run |

**Exit codes:** 0 = all pass, 1 = one or more suites exceed threshold, 2 = setup error.
Human-readable summary → stderr. XML → stdout or `-output` file.

## Results (5-run medians, Linux/amd64, baseline.xml)

Startup overhead: **3.6 ms** per invocation (binary cold-start).

| Suite | GNU | aict `--plain` | ratio | aict `--xml` | ratio | Pass |
|-------|-----|----------------|-------|--------------|-------|------|
| `diff` | 0.9 ms | 1.9 ms | 2.1x | 2.1 ms | 2.4x | ✅ |
| `wc` | 6.1 ms | 16.0 ms | 2.6x | 17.0 ms | 2.7x | ✅ |
| `awk` | 4.1 ms | 12.0 ms | 2.9x | 11.0 ms | 2.6x | ✅ |
| `sed` | 3.3 ms | 13.9 ms | 4.2x | 16.1 ms | 4.9x | ✅ |
| `find` | 1.9 ms | 13.1 ms | 6.8x | 15.3 ms | 8.0x | ✅ |
| `ls` | 4.0 ms | 51.1 ms | 12.9x | 70.3 ms | 17.7x | ❌ |
| `cat` | 1.4 ms | 23.7 ms | 16.4x | 31.3 ms | 21.6x | ❌ |
| `grep` | 1.3 ms | 119.2 ms | 88.4x | 129.7 ms | 96.2x | ❌ |

Pass criterion: **< 10× slower** than the GNU baseline (median).

## Root cause analysis

### grep (88–96×): Go regexp vs SIMD search

GNU `grep` uses Boyer-Moore-Horspool and SIMD instructions. Go's `regexp`
package is a general-purpose NFA engine. Even with `-F` (fixed strings), Go
routes through `regexp.QuoteMeta` and still pays NFA overhead.

The `BenchmarkGrep_StringsContains_100k` benchmark shows what a raw
`strings.Contains` scan costs (~8 ms) vs the regexp path (~45 ms). A literal
fast-path (detect patterns with no regex metacharacters → use
`strings.Contains`) would recover ~5× on literal searches.

### cat (16–22×): line-by-line scanning

GNU `cat` is a streaming `sendfile`/`read` loop. aict's `readFileContent`
uses `bufio.Scanner.Scan()` which allocates a string per line and builds a
`[]string` slice in memory. For 100k lines that is 100k allocations before
XML marshaling.

Switching to a single `os.ReadFile` + `bytes.Count` for line counting would
close most of the gap.

### ls (13–18×): MIME and language detection per file

`buildEntry` in `ls.go` opens each file twice — once for
`detect.DetectFromFile()` (512-byte read → `http.DetectContentType()`) and
once for `detect.LanguageFromFile()` (reads first line for shebang). For 1000
files that is 2000 extra `open/read/close` syscalls. Plain mode (`cfg.Plain`)
skips both, which the `BenchmarkLS_Plain_1000` benchmark verifies.

## Regression workflow

```bash
# 1. Capture current state as baseline
make bench-baseline            # writes benchmarks/baseline.xml

# 2. Make your change

# 3. Compare
make bench-compare
# <delta> elements show plain_delta_pct and xml_delta_pct
# Regressions > 5% are flagged with regression="true"
```

## Go testing.B benchmarks

These call tool-internal functions directly (no subprocess overhead) and are
compatible with `benchstat`:

```
tools/grep/bench_test.go
  BenchmarkGrep_Regex_100k           — regexp path, 100k-line file
  BenchmarkGrep_Literal_100k         — fixed-string path (regexp.QuoteMeta)
  BenchmarkGrep_StringsContains_100k — baseline: raw strings.Contains scan
  BenchmarkGrep_Recursive_1000       — searchDirectory over 1000 small files

tools/cat/bench_test.go
  BenchmarkCat_Stream_100k           — readFileContent on 100k-line file
  BenchmarkCat_BinaryDetect          — binary detection scan in isolation
  BenchmarkCat_SmallFile             — single small Go source file

tools/ls/bench_test.go
  BenchmarkLS_WithDetection_1000     — MIME + language detection per file
  BenchmarkLS_Plain_1000             — plain mode (no detection)
```

To compare two implementations with `benchstat`:

```bash
# Install benchstat once:
go install golang.org/x/perf/cmd/benchstat@latest

# Capture before:
go test -bench=. -count=10 ./tools/grep/ | tee old.txt

# Make your change, then capture after:
go test -bench=. -count=10 ./tools/grep/ | tee new.txt

benchstat old.txt new.txt
```
