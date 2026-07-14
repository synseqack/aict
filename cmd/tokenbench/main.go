// tokenbench — measure the context-window cost of aict output vs GNU coreutils.
//
// Speed benchmarks (cmd/bench) answer "how fast does it run"; this answers the
// question aict actually exists for: "how many tokens does an agent spend to
// complete a task". For each scenario we capture the full command transcript —
// including the follow-up calls a plaintext workflow forces (file(1) for
// language, stat(1) for metadata) — for both the GNU toolchain and aict, over
// an identical fixture tree.
//
// Usage:
//
//	go build -o aict . && go run ./cmd/tokenbench -aict ./aict -samples benchmarks/token-samples
//
// Flags:
//
//	-aict PATH      path to aict binary, default ./aict
//	-samples DIR    write raw captured outputs here (default: none)
//	-markdown FILE  write a markdown summary table (default: stdout summary only)
//
// Token counts here are a chars/4 estimate. For real tokenizer counts, run
// benchmarks/count_tokens.py over the samples directory (uses tiktoken).
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type capture struct {
	Name  string // e.g. "inventory-gnu"
	Text  string
	Calls int
}

type scenario struct {
	Name     string
	Question string // the agent task this transcript answers
	GNU      func(dir string) (capture, error)
	AICT     func(dir string, aictBin string) (capture, error)
}

func main() {
	aictFlag := flag.String("aict", "./aict", "path to aict binary")
	samplesFlag := flag.String("samples", "", "directory to write raw captured outputs")
	markdownFlag := flag.String("markdown", "", "write markdown summary table to file")
	flag.Parse()

	aictBin, err := filepath.Abs(*aictFlag)
	if err != nil || !exists(aictBin) {
		fmt.Fprintf(os.Stderr, "tokenbench: aict binary not found at %q\n  build it first: go build -o aict .\n", *aictFlag)
		os.Exit(2)
	}

	for _, tool := range []string{"ls", "file", "stat", "grep", "find", "cat", "wc", "diff"} {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Fprintf(os.Stderr, "tokenbench: required GNU tool %q not on PATH\n", tool)
			os.Exit(2)
		}
	}

	fmt.Fprintln(os.Stderr, "tokenbench: preparing fixture tree…")
	dir, cleanup, err := setupFixtures()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tokenbench: setup failed: %v\n", err)
		os.Exit(2)
	}
	defer cleanup()

	var rows []string
	rows = append(rows,
		"| Task | GNU calls | GNU tokens* | aict calls | aict tokens* | Ratio |",
		"|------|-----------|-------------|------------|--------------|-------|")

	for _, sc := range scenarios() {
		gnu, err := sc.GNU(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tokenbench: %s (gnu): %v\n", sc.Name, err)
			os.Exit(2)
		}
		aict, err := sc.AICT(dir, aictBin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tokenbench: %s (aict): %v\n", sc.Name, err)
			os.Exit(2)
		}

		if *samplesFlag != "" {
			if err := writeSample(*samplesFlag, sc.Name+"-gnu.txt", gnu.Text); err != nil {
				fmt.Fprintf(os.Stderr, "tokenbench: %v\n", err)
				os.Exit(2)
			}
			if err := writeSample(*samplesFlag, sc.Name+"-aict.txt", aict.Text); err != nil {
				fmt.Fprintf(os.Stderr, "tokenbench: %v\n", err)
				os.Exit(2)
			}
		}

		gnuTok := estTokens(gnu.Text)
		aictTok := estTokens(aict.Text)
		ratio := float64(aictTok) / float64(max(gnuTok, 1))
		rows = append(rows, fmt.Sprintf("| %s — %s | %d | %d | %d | %d | %.2fx |",
			sc.Name, sc.Question, gnu.Calls, gnuTok, aict.Calls, aictTok, ratio))

		fmt.Fprintf(os.Stderr, "  %-10s gnu: %d calls / ~%d tok   aict: %d calls / ~%d tok\n",
			sc.Name, gnu.Calls, gnuTok, aict.Calls, aictTok)
	}

	table := strings.Join(rows, "\n") + "\n\n\\* chars/4 estimate — run benchmarks/count_tokens.py for real tokenizer counts.\n"
	if *markdownFlag != "" {
		if err := os.WriteFile(*markdownFlag, []byte(table), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "tokenbench: %v\n", err)
			os.Exit(2)
		}
	} else {
		fmt.Println(table)
	}
}

func scenarios() []scenario {
	return []scenario{
		{
			Name:     "inventory",
			Question: "list a directory with size, type and language per entry",
			// Plaintext ls has no language/MIME column, so the agent issues a
			// follow-up file(1) call to classify the entries.
			GNU: func(dir string) (capture, error) {
				src := filepath.Join(dir, "src")
				out1, err := runCmd(src, "ls", "-la")
				if err != nil {
					return capture{}, err
				}
				entries, err := os.ReadDir(src)
				if err != nil {
					return capture{}, err
				}
				args := []string{}
				for _, e := range entries {
					if !e.IsDir() {
						args = append(args, e.Name())
					}
				}
				out2, err := runCmd(src, "file", args...)
				if err != nil {
					return capture{}, err
				}
				return capture{Text: out1 + out2, Calls: 2}, nil
			},
			AICT: func(dir, aictBin string) (capture, error) {
				out, err := runCmd(dir, aictBin, "ls", "src", "--xml")
				return capture{Text: out, Calls: 1}, err
			},
		},
		{
			Name:     "read",
			Question: "read a source file plus its line count and type",
			GNU: func(dir string) (capture, error) {
				target := filepath.Join("src", "server.go")
				out1, err := runCmd(dir, "cat", "-n", target)
				if err != nil {
					return capture{}, err
				}
				out2, err := runCmd(dir, "wc", target)
				if err != nil {
					return capture{}, err
				}
				out3, err := runCmd(dir, "file", target)
				if err != nil {
					return capture{}, err
				}
				return capture{Text: out1 + out2 + out3, Calls: 3}, nil
			},
			AICT: func(dir, aictBin string) (capture, error) {
				out, err := runCmd(dir, aictBin, "cat", filepath.Join("src", "server.go"), "-n", "--xml")
				return capture{Text: out, Calls: 1}, err
			},
		},
		{
			Name:     "search",
			Question: "find a pattern with file, line and context",
			GNU: func(dir string) (capture, error) {
				out, err := runCmd(dir, "grep", "-rn", "-C1", "TODO", "src")
				return capture{Text: out, Calls: 1}, err
			},
			AICT: func(dir, aictBin string) (capture, error) {
				out, err := runCmd(dir, aictBin, "grep", "TODO", "src", "-r", "-n", "-C", "1", "--xml")
				return capture{Text: out, Calls: 1}, err
			},
		},
		{
			Name:     "locate",
			Question: "find all .go files with size and mtime",
			// find prints bare paths; the agent needs one stat call per file
			// to get size/mtime. aict find returns both in one shot.
			GNU: func(dir string) (capture, error) {
				out, err := runCmd(dir, "find", "src", "-name", "*.go")
				if err != nil {
					return capture{}, err
				}
				calls := 1
				var b strings.Builder
				b.WriteString(out)
				for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
					if line == "" {
						continue
					}
					sout, err := runCmd(dir, "stat", "-c", "%n %s %Y", line)
					if err != nil {
						return capture{}, err
					}
					b.WriteString(sout)
					calls++
				}
				return capture{Text: b.String(), Calls: calls}, nil
			},
			AICT: func(dir, aictBin string) (capture, error) {
				out, err := runCmd(dir, aictBin, "find", "src", "-name", "*.go", "--xml")
				return capture{Text: out, Calls: 1}, err
			},
		},
		{
			Name:     "compare",
			Question: "diff two files with change types and line numbers",
			GNU: func(dir string) (capture, error) {
				// diff exits 1 when files differ; that's expected here
				out, _ := runCmd(dir, "diff", "-u", "config_old.yaml", "config_new.yaml")
				return capture{Text: out, Calls: 1}, nil
			},
			AICT: func(dir, aictBin string) (capture, error) {
				out, err := runCmd(dir, aictBin, "diff", "config_old.yaml", "config_new.yaml", "--xml")
				return capture{Text: out, Calls: 1}, err
			},
		},
	}
}

func runCmd(workDir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		if len(out) > 0 {
			return string(out), nil
		}
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return string(out), nil
}

func estTokens(s string) int {
	return len(s) / 4
}

func writeSample(dir, name, text string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(text), 0644)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
