// bench — compare aict tools against GNU coreutils equivalents.
//
// Usage:
//
//	go build -o bench ./cmd/bench && ./bench [flags]
//	go install ./cmd/bench
//
// Flags:
//
//	-runs N         number of runs per suite (median reported), default 7
//	-output FILE    write XML results to FILE instead of stdout
//	-compare FILE   load a baseline XML and emit <delta> per suite
//	-threshold F    pass/fail ratio vs GNU baseline, default 10.0
//	-aict PATH      path to aict binary, default ./aict
//	-suites LIST    comma-separated suite names to run (default: all)
//
// Exit codes: 0 all pass, 1 one or more suites exceed threshold, 2 setup error.
// Human-readable summary is written to stderr; XML goes to stdout (or -output).
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

func main() {
	runsFlag := flag.Int("runs", 7, "number of runs per suite (median reported)")
	outputFlag := flag.String("output", "", "write XML results to file (default: stdout)")
	compareFlag := flag.String("compare", "", "compare against a baseline XML file")
	thresholdFlag := flag.Float64("threshold", 10.0, "pass/fail ratio threshold vs GNU baseline")
	aictFlag := flag.String("aict", "./aict", "path to aict binary")
	suitesFlag := flag.String("suites", "", "comma-separated suite names to run (default: all)")
	flag.Parse()

	aictBin := *aictFlag
	if _, err := os.Stat(aictBin); err != nil {
		fmt.Fprintf(os.Stderr, "bench: aict binary not found at %q\n  build it first: go build -o aict .\n", aictBin)
		os.Exit(2)
	}

	// Suite filter
	var suiteFilter map[string]bool
	if *suitesFlag != "" {
		suiteFilter = make(map[string]bool)
		for _, s := range strings.Split(*suitesFlag, ",") {
			suiteFilter[strings.TrimSpace(s)] = true
		}
	}

	// Test data
	fmt.Fprintln(os.Stderr, "bench: preparing test data…")
	dir, cleanup, err := setupTestData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bench: setup failed: %v\n", err)
		os.Exit(2)
	}
	defer cleanup()

	// Baseline for comparison
	var baseline *BenchmarkReport
	if *compareFlag != "" {
		baseline, err = loadBaseline(*compareFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bench: cannot load baseline %q: %v\n", *compareFlag, err)
			os.Exit(2)
		}
	}

	// Startup overhead
	fmt.Fprintln(os.Stderr, "bench: measuring startup overhead…")
	startupMs := measureStartup(aictBin, *runsFlag)

	// Suites
	fmt.Fprintln(os.Stderr, "bench: running suites…")
	suites := buildSuites(dir, aictBin, *runsFlag, *thresholdFlag)

	if suiteFilter != nil {
		var filtered []Suite
		for _, s := range suites {
			if suiteFilter[s.Name] {
				filtered = append(filtered, s)
			}
		}
		suites = filtered
	}

	// Build report
	report := BenchmarkReport{
		Runs:      *runsFlag,
		Timestamp: time.Now().Unix(),
		StartupMs: startupMs,
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		Threshold: *thresholdFlag,
		Suites:    suites,
	}

	if baseline != nil {
		addDeltas(&report, baseline, *compareFlag)
	}

	// Output
	out := os.Stdout
	if *outputFlag != "" {
		f, err := os.Create(*outputFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bench: cannot create output file: %v\n", err)
			os.Exit(2)
		}
		defer f.Close()
		out = f
	}

	if err := writeReport(out, &report); err != nil {
		fmt.Fprintf(os.Stderr, "bench: write failed: %v\n", err)
		os.Exit(2)
	}

	printSummary(os.Stderr, &report)

	// Exit code
	for _, s := range report.Suites {
		if !s.Pass {
			os.Exit(1)
		}
	}
}
