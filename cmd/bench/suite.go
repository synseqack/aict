package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── XML schema ─────────────────────────────────────────────────────────────

type BenchmarkReport struct {
	XMLName   xml.Name `xml:"benchmark"`
	Runs      int      `xml:"runs,attr"`
	Timestamp int64    `xml:"timestamp,attr"`
	StartupMs float64  `xml:"startup_ms,attr"`
	GoVersion string   `xml:"go_version,attr"`
	GOOS      string   `xml:"goos,attr"`
	GOARCH    string   `xml:"goarch,attr"`
	Threshold float64  `xml:"threshold,attr"`
	Suites    []Suite  `xml:"suite"`
}

type Suite struct {
	Name        string    `xml:"name,attr"`
	Description string    `xml:"description,attr"`
	Pass        bool      `xml:"pass,attr"`
	Baseline    Timing    `xml:"baseline"`
	Plain       Variant   `xml:"plain"`
	XML         Variant   `xml:"xml"`
	Delta       *Delta    `xml:"delta,omitempty"`
	Samples     []Sample  `xml:"observations>sample"`
}

type Timing struct {
	Cmd      string  `xml:"cmd,attr"`
	MinMs    float64 `xml:"min_ms,attr"`
	MedianMs float64 `xml:"median_ms,attr"`
	MaxMs    float64 `xml:"max_ms,attr"`
}

type Variant struct {
	Mode     string  `xml:"mode,attr"`
	MinMs    float64 `xml:"min_ms,attr"`
	MedianMs float64 `xml:"median_ms,attr"`
	MaxMs    float64 `xml:"max_ms,attr"`
	Ratio    float64 `xml:"ratio,attr"`
	Pass     bool    `xml:"pass,attr"`
	Note     string  `xml:"note,attr,omitempty"`
}

type Delta struct {
	Vs            string  `xml:"vs,attr"`
	PlainDeltaPct float64 `xml:"plain_delta_pct,attr"`
	XMLDeltaPct   float64 `xml:"xml_delta_pct,attr"`
	Regression    bool    `xml:"regression,attr"`
}

type Sample struct {
	Index      int     `xml:"index,attr"`
	BaselineMs float64 `xml:"baseline_ms,attr"`
	PlainMs    float64 `xml:"plain_ms,attr"`
	XMLMs      float64 `xml:"xml_ms,attr"`
}

// ── suite definitions ──────────────────────────────────────────────────────

type suiteSpec struct {
	name, desc, note string
	base, plain, xml []string
}

func buildSuites(dir, aictBin string, runs int, threshold float64) []Suite {
	largeFile := filepath.Join(dir, "large_file.txt")
	csvFile := filepath.Join(dir, "data.csv")
	f1 := filepath.Join(dir, "f1.txt")
	f2 := filepath.Join(dir, "f2.txt")

	specs := []suiteSpec{
		{
			name: "ls", desc: "1000 Go source files",
			note: "aict adds MIME/language detection per file",
			base:  []string{"ls", dir},
			plain: []string{aictBin, "ls", dir, "--plain"},
			xml:   []string{aictBin, "ls", dir, "--xml"},
		},
		{
			name: "grep", desc: "pattern match across 100k-line file",
			note: "GNU grep uses SIMD/Boyer-Moore; Go regexp is general-purpose",
			base:  []string{"grep", "search", largeFile},
			plain: []string{aictBin, "grep", "search", largeFile, "--plain"},
			xml:   []string{aictBin, "grep", "search", largeFile, "--xml"},
		},
		{
			name: "cat", desc: "stream 100k-line file",
			note: "aict scans lines + detects binary/encoding; GNU cat is a raw copy",
			base:  []string{"cat", largeFile},
			plain: []string{aictBin, "cat", largeFile, "--plain"},
			xml:   []string{aictBin, "cat", largeFile, "--xml"},
		},
		{
			name: "find", desc: "walk deep directory tree, filter *.go",
			base:  []string{"find", dir, "-name", "*.go"},
			plain: []string{aictBin, "find", dir, "-name", "*.go", "--plain"},
			xml:   []string{aictBin, "find", dir, "-name", "*.go", "--xml"},
		},
		{
			name: "diff", desc: "1000-line files with 1 added line",
			note: "Myers O(ND) algorithm; comparable to GNU diff",
			base:  []string{"diff", f1, f2},
			plain: []string{aictBin, "diff", f1, f2, "--plain"},
			xml:   []string{aictBin, "diff", f1, f2, "--xml"},
		},
		{
			name: "sed", desc: "global substitution on 10k-line CSV",
			base:  []string{"sed", "s/user/USER/g", csvFile},
			plain: []string{aictBin, "sed", "-e", "s/user/USER/g", csvFile, "--plain"},
			xml:   []string{aictBin, "sed", "-e", "s/user/USER/g", csvFile, "--xml"},
		},
		{
			name: "awk", desc: "first-field extraction on 10k-line CSV",
			base:  []string{"awk", "-F,", "{print $1}", csvFile},
			plain: []string{aictBin, "awk", "-F,", "{print $1}", csvFile, "--plain"},
			xml:   []string{aictBin, "awk", "-F,", "{print $1}", csvFile, "--xml"},
		},
		{
			name: "wc", desc: "count lines/words/bytes in 100k-line file",
			base:  []string{"wc", largeFile},
			plain: []string{aictBin, "wc", largeFile, "--plain"},
			xml:   []string{aictBin, "wc", largeFile, "--xml"},
		},
	}

	suites := make([]Suite, 0, len(specs))
	for _, sp := range specs {
		fmt.Fprintf(os.Stderr, "  %-6s ", sp.name)
		s := measureSuite(sp, runs, threshold)
		suites = append(suites, s)
		status := "✅"
		if !s.Pass {
			status = "❌"
		}
		fmt.Fprintf(os.Stderr, "%s plain=%.1fms (%.1fx) xml=%.1fms (%.1fx)\n",
			status, s.Plain.MedianMs, s.Plain.Ratio, s.XML.MedianMs, s.XML.Ratio)
	}
	return suites
}

// ── measurement ────────────────────────────────────────────────────────────

type rawSample struct{ baseline, plain, xml time.Duration }

func measureSuite(sp suiteSpec, runs int, threshold float64) Suite {
	samples := make([]rawSample, runs)
	for i := range samples {
		samples[i].baseline = runCmd(sp.base[0], sp.base[1:]...)
		samples[i].plain = runCmd(sp.plain[0], sp.plain[1:]...)
		samples[i].xml = runCmd(sp.xml[0], sp.xml[1:]...)
	}

	baseTimes := extract(samples, func(s rawSample) time.Duration { return s.baseline })
	plainTimes := extract(samples, func(s rawSample) time.Duration { return s.plain })
	xmlTimes := extract(samples, func(s rawSample) time.Duration { return s.xml })

	sortDurations(baseTimes)
	sortDurations(plainTimes)
	sortDurations(xmlTimes)

	baseMedian := ms(baseTimes[len(baseTimes)/2])
	plainMedian := ms(plainTimes[len(plainTimes)/2])
	xmlMedian := ms(xmlTimes[len(xmlTimes)/2])

	plainRatio := ratio(plainMedian, baseMedian)
	xmlRatio := ratio(xmlMedian, baseMedian)

	suite := Suite{
		Name:        sp.name,
		Description: sp.desc,
		Pass:        plainRatio < threshold && xmlRatio < threshold,
		Baseline: Timing{
			Cmd:      sp.base[0],
			MinMs:    ms(baseTimes[0]),
			MedianMs: baseMedian,
			MaxMs:    ms(baseTimes[len(baseTimes)-1]),
		},
		Plain: Variant{
			Mode:     "plain",
			MinMs:    ms(plainTimes[0]),
			MedianMs: plainMedian,
			MaxMs:    ms(plainTimes[len(plainTimes)-1]),
			Ratio:    plainRatio,
			Pass:     plainRatio < threshold,
			Note:     sp.note,
		},
		XML: Variant{
			Mode:     "xml",
			MinMs:    ms(xmlTimes[0]),
			MedianMs: xmlMedian,
			MaxMs:    ms(xmlTimes[len(xmlTimes)-1]),
			Ratio:    xmlRatio,
			Pass:     xmlRatio < threshold,
		},
	}

	suite.Samples = make([]Sample, runs)
	for i, s := range samples {
		suite.Samples[i] = Sample{
			Index:      i,
			BaselineMs: ms(s.baseline),
			PlainMs:    ms(s.plain),
			XMLMs:      ms(s.xml),
		}
	}

	return suite
}

func measureStartup(aictBin string, runs int) float64 {
	times := make([]time.Duration, runs)
	for i := range times {
		times[i] = runCmd(aictBin, "help")
	}
	sortDurations(times)
	return ms(times[len(times)/2])
}

func runCmd(cmd string, args ...string) time.Duration {
	start := time.Now()
	exec.Command(cmd, args...).Run()
	return time.Since(start)
}

// ── baseline comparison ────────────────────────────────────────────────────

func loadBaseline(path string) (*BenchmarkReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r BenchmarkReport
	if err := xml.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func addDeltas(report *BenchmarkReport, baseline *BenchmarkReport, baselinePath string) {
	baseMap := make(map[string]Suite, len(baseline.Suites))
	for _, s := range baseline.Suites {
		baseMap[s.Name] = s
	}

	vsLabel := filepath.Base(baselinePath)

	for i, s := range report.Suites {
		b, ok := baseMap[s.Name]
		if !ok {
			continue
		}
		plainDelta := pctDelta(s.Plain.MedianMs, b.Plain.MedianMs)
		xmlDelta := pctDelta(s.XML.MedianMs, b.XML.MedianMs)
		regression := math.Abs(plainDelta) > 5 || math.Abs(xmlDelta) > 5
		report.Suites[i].Delta = &Delta{
			Vs:            vsLabel,
			PlainDeltaPct: plainDelta,
			XMLDeltaPct:   xmlDelta,
			Regression:    regression,
		}
	}
}

func pctDelta(current, baseline float64) float64 {
	if baseline == 0 {
		return 0
	}
	return ((current - baseline) / baseline) * 100
}

// ── output ──────────────────────────────────────────────────────────────────

func writeReport(w io.Writer, report *BenchmarkReport) error {
	fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	out, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(out))
	return err
}

func printSummary(w io.Writer, report *BenchmarkReport) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "bench: %s %s/%s  startup=%.1fms  runs=%d  threshold=%.0fx\n",
		report.GoVersion, report.GOOS, report.GOARCH,
		report.StartupMs, report.Runs, report.Threshold)
	fmt.Fprintln(w)

	passCount := 0
	fmt.Fprintf(w, "  %-8s %-32s %8s %8s  %8s %8s  %s\n",
		"suite", "description", "plain", "(ratio)", "xml", "(ratio)", "")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 85))

	for _, s := range report.Suites {
		mark := "✅"
		if !s.Pass {
			mark = "❌"
		} else {
			passCount++
		}
		deltaLine := ""
		if s.Delta != nil {
			deltaLine = fmt.Sprintf("Δplain=%+.1f%% Δxml=%+.1f%%", s.Delta.PlainDeltaPct, s.Delta.XMLDeltaPct)
			if s.Delta.Regression {
				deltaLine += " ⚠ REGRESSION"
			}
		}
		fmt.Fprintf(w, "  %-8s %-32s %6.1fms %6.1fx   %6.1fms %6.1fx  %s %s\n",
			s.Name,
			truncate(s.Description, 32),
			s.Plain.MedianMs, s.Plain.Ratio,
			s.XML.MedianMs, s.XML.Ratio,
			mark, deltaLine,
		)
	}

	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 85))
	fmt.Fprintf(w, "  %d/%d suites within %.0fx threshold\n", passCount, len(report.Suites), report.Threshold)
	fmt.Fprintln(w)
}

// ── helpers ────────────────────────────────────────────────────────────────

func extract(samples []rawSample, fn func(rawSample) time.Duration) []time.Duration {
	out := make([]time.Duration, len(samples))
	for i, s := range samples {
		out[i] = fn(s)
	}
	return out
}

func sortDurations(d []time.Duration) {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000.0 }

func ratio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
