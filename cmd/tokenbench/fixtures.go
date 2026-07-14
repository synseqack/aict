package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// setupFixtures builds a small but realistic project tree: mixed languages,
// nested dirs, a TODO-annotated source set, and a pair of near-identical
// configs. Sizes are kept modest so captured transcripts stay representative
// of a normal agent interaction, not a pathological one.
func setupFixtures() (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "aict-tokenbench-")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(dir) }

	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(src, "internal"), 0755); err != nil {
		cleanup()
		return "", nil, err
	}

	files := map[string]string{
		"server.go": `package main

import (
	"fmt"
	"net/http"
)

// TODO: add graceful shutdown
func main() {
	http.HandleFunc("/", handler)
	fmt.Println("listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	// TODO: validate request method
	fmt.Fprintln(w, "ok")
}
`,
		"client.go": `package main

import "net/http"

func fetch(url string) (*http.Response, error) {
	return http.Get(url)
}
`,
		"util.py": `import os

# TODO: handle missing env vars
def load_config():
    return {k: v for k, v in os.environ.items() if k.startswith("APP_")}
`,
		"app.js": `const express = require('express');
const app = express();
// TODO: move port to config
app.listen(3000);
`,
		"README.md": `# Example service

A small demo service used as a token-benchmark fixture.
`,
		filepath.Join("internal", "helpers.go"): `package internal

func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(src, name), []byte(content), 0644); err != nil {
			cleanup()
			return "", nil, err
		}
	}

	// Two near-identical configs for the diff scenario
	var oldCfg, newCfg strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&oldCfg, "key_%d: value_%d\n", i, i)
		if i == 20 {
			fmt.Fprintf(&newCfg, "key_%d: CHANGED\n", i)
		} else {
			fmt.Fprintf(&newCfg, "key_%d: value_%d\n", i, i)
		}
	}
	newCfg.WriteString("added_key: added_value\n")

	if err := os.WriteFile(filepath.Join(dir, "config_old.yaml"), []byte(oldCfg.String()), 0644); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "config_new.yaml"), []byte(newCfg.String()), 0644); err != nil {
		cleanup()
		return "", nil, err
	}

	return dir, cleanup, nil
}
