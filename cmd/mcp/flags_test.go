package mcpserver

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/synseqack/aict/internal/tool"
)

// TestFlagMappingsMatchSchema pins the MCP flag table to reality. A mapping
// whose key is neither a schema property nor a declared positional is dead —
// it can never be reached from an MCP call. A schema property with no mapping
// and no positional entry is unreachable for the same reason. Both regress
// silently, since buildArgs drops anything it cannot translate.
func TestFlagMappingsMatchSchema(t *testing.T) {
	// The universal output flags are accepted by every tool but advertised
	// by none of them, so they are exempt from the reachability check.
	exempt := map[string]bool{"nocompact": true, "pretty": true, "dict": true}

	var bad []string
	names := make([]string, 0, len(tool.AllMeta()))
	for name := range tool.AllMeta() {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		meta, ok := tool.GetMeta(name)
		if !ok {
			bad = append(bad, fmt.Sprintf("%s: registered in flagMappings but no tool meta", name))
			continue
		}
		props, _ := meta.InputSchema["properties"].(map[string]interface{})
		if props == nil {
			props = map[string]interface{}{}
		}

		// Every flag mapping key must be a real schema property; a key
		// that is not is a typo that silently disables the flag.
		keys := make([]string, 0, len(flagMappings[name]))
		for k := range flagMappings[name] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, ok := props[k]; !ok {
				bad = append(bad, fmt.Sprintf("%s.%s: not in input schema (dead mapping)", name, k))
			}
		}

		// Every advertised property must be reachable: either a flag or a
		// declared positional. Otherwise an agent can set it and have it
		// discarded without diagnostic.
		for p := range props {
			if exempt[p] {
				continue
			}
			_, mapped := flagMappings[name][p]
			_, positional := positionalProperty(name, p)
			if !mapped && !positional {
				bad = append(bad, fmt.Sprintf("%s.%s: advertised but neither mapped nor positional (unreachable)", name, p))
			}
		}
	}
	if len(bad) > 0 {
		t.Fatalf("flag table inconsistent with input schema:\n  %s", strings.Join(bad, "\n  "))
	}
}

// TestPositionalsAreAdvertised checks the reverse direction: after
// mergePositionalSchema runs, every declared positional is present in the
// schema with the right type, so an agent can discover how to pass one.
func TestPositionalsAreAdvertised(t *testing.T) {
	names := make([]string, 0, len(positionalInputs))
	for name := range positionalInputs {
		names = append(names, name)
	}
	sort.Strings(names)

	var bad []string
	for _, name := range names {
		meta, ok := tool.GetMeta(name)
		if !ok {
			bad = append(bad, fmt.Sprintf("%s: no tool meta", name))
			continue
		}
		schema := map[string]interface{}{}
		for k, v := range meta.InputSchema {
			schema[k] = v
		}
		mergePositionalSchema(name, schema)

		props, _ := schema["properties"].(map[string]interface{})
		for _, p := range positionalOrder(name) {
			got, exists := props[p.property]
			if !exists {
				bad = append(bad, fmt.Sprintf("%s.%s: not advertised after merge", name, p.property))
				continue
			}
			prop, _ := got.(map[string]interface{})
			want := "string"
			if p.array {
				want = "array"
			}
			if got := prop["type"]; got != want {
				bad = append(bad, fmt.Sprintf("%s.%s: type %v, want %s", name, p.property, got, want))
			}
		}
	}
	if len(bad) > 0 {
		t.Fatalf("positional table inconsistent:\n  %s", strings.Join(bad, "\n  "))
	}
}

// TestBuildArgsDeterministic pins the property that motivated the rewrite:
// the same argument object must always yield the same argv. JSON objects
// arrive with no order, so a map walk shuffled positionals and reversed
// `diff a b` on roughly one call in five.
func TestBuildArgsDeterministic(t *testing.T) {
	cases := []struct {
		tool string
		args map[string]interface{}
	}{
		{"diff", map[string]interface{}{"old": "a.txt", "new": "b.txt", "quiet": true}},
		{"cat", map[string]interface{}{"files": []interface{}{"f1", "f2", "f3"}}},
		{"tr", map[string]interface{}{"set1": "a-z", "set2": "A-Z", "delete": true}},
		{"grep", map[string]interface{}{"pattern": "func", "root": "src", "recursive": true, "linenumbers": true}},
		{"basename", map[string]interface{}{"names": []interface{}{"/a/b.go", "/c/d.go"}, "suffix": ".go"}},
	}
	for _, c := range cases {
		first, err := buildArgs(c.tool, c.args)
		if err != nil {
			t.Fatalf("%s: %v", c.tool, err)
		}
		for i := 0; i < 50; i++ {
			got, err := buildArgs(c.tool, c.args)
			if err != nil {
				t.Fatalf("%s: %v", c.tool, err)
			}
			if len(got) != len(first) {
				t.Fatalf("%s: argv length drift %v vs %v", c.tool, first, got)
			}
			for j := range got {
				if got[j] != first[j] {
					t.Fatalf("%s: argv differs on call %d:\n  %v\n  %v", c.tool, i, first, got)
				}
			}
		}
	}
}

// TestBuildArgsPositionalOrder asserts the specific orderings that a
// shuffled argv would break outright (reversed operands, wrong pattern
// position) rather than merely render nondeterministically.
func TestBuildArgsPositionalOrder(t *testing.T) {
	got, err := buildArgs("diff", map[string]interface{}{"old": "a.txt", "new": "b.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a.txt" || got[1] != "b.txt" {
		t.Fatalf("diff positionals out of order: %v", got)
	}

	got, err = buildArgs("grep", map[string]interface{}{"pattern": "func", "root": "src"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "func" || got[1] != "src" {
		t.Fatalf("grep pattern must precede root: %v", got)
	}

	got, err = buildArgs("tr", map[string]interface{}{"set1": "a-z", "set2": "A-Z"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a-z" || got[1] != "A-Z" {
		t.Fatalf("tr sets out of order: %v", got)
	}
}

// TestBuildArgsArrayExpansion covers the []interface{} case: JSON arrays were
// silently dropped by the old type switch, so `checksums -a md5 -a sha256`
// was unreachable over MCP.
func TestBuildArgsArrayExpansion(t *testing.T) {
	got, err := buildArgs("checksums", map[string]interface{}{
		"algorithms": []interface{}{"md5", "sha256"},
		"files":      []interface{}{"a.txt", "b.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// -a md5 -a sha256, then the two files positionally.
	want := []string{"-a", "md5", "-a", "sha256", "a.txt", "b.txt"}
	if len(got) != len(want) {
		t.Fatalf("checksums argv: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("checksums argv: %v", got)
		}
	}
}

// TestBuildArgsZeroInteger guards the regression where an integer property
// of zero was skipped: `head -n 0` is a real request, not an omitted one.
func TestBuildArgsZeroInteger(t *testing.T) {
	got, err := buildArgs("head", map[string]interface{}{"lines": float64(0), "files": []interface{}{"a.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-n", "0", "a.txt"}
	if len(got) != len(want) {
		t.Fatalf("head argv: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("head argv: %v", got)
		}
	}
}

// TestAnnotateWithLegend checks the compact-name legend is valid JSON, stays
// proportional to the response, and leaves non-JSON output untouched.
func TestAnnotateWithLegend(t *testing.T) {
	full := annotateWithLegend("ls", `{"p":"/x","n":3}`)
	var decoded struct {
		Legend  map[string]string `json:"_legend"`
		P       string            `json:"p"`
		Entries int               `json:"n"`
	}
	if err := json.Unmarshal([]byte(full), &decoded); err != nil {
		t.Fatalf("annotated output is not valid JSON: %v\ngot: %s", err, full)
	}
	if decoded.Legend["p"] != "path" {
		t.Errorf("legend[p] = %q, want %q", decoded.Legend["p"], "path")
	}
	if decoded.P != "/x" || decoded.Entries != 3 {
		t.Errorf("payload altered: %+v", decoded)
	}
	// Only abbreviations the response actually used may appear: n and p are
	// present, ent is not.
	for short := range decoded.Legend {
		if short != "p" && short != "n" {
			t.Errorf("legend includes %q, which the response never used", short)
		}
	}

	// An object with no payload of its own must still parse.
	var empty map[string]interface{}
	if err := json.Unmarshal([]byte(annotateWithLegend("ls", `{}`)), &empty); err != nil {
		t.Fatalf("empty object produced invalid JSON: %v", err)
	}

	// A response that shares no keys with the dict passes through untouched.
	none := annotateWithLegend("ls", `{"zzz":"no such key"}`)
	if none != `{"zzz":"no such key"}` {
		t.Errorf("unmatched response altered: %s", none)
	}

	// Non-JSON output must pass through untouched.
	plain := annotateWithLegend("ls", "<dict><p>path</p></dict>")
	if plain != "<dict><p>path</p></dict>" {
		t.Errorf("non-JSON output altered: %s", plain)
	}
}
