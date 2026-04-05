# AGENTS.md — AI-Coreutils Development Guide

> Instructions for AI coding agents implementing `aict`.
> Read this before writing any code.

---

## Project Overview

**What**: Single Go binary (`aict`) reimplementing ~22 Unix CLI tools that AI coding agents actually call, outputting structured XML instead of plaintext.

**Why**: AI agents waste tokens parsing human-readable `ls`, `grep`, `cat` output. XML output carries 3× semantic density with zero ambiguity.

**Constraints**:
- Go only, zero non-stdlib dependencies
- Single binary, subcommand model: `aict ls src/`
- Three output modes: XML (default for AI), JSON, plain text (compatibility)
- All paths absolute in output
- All timestamps Unix epoch integers
- All sizes in bytes
- Structured errors, never stderr
- Empty results are valid XML

---

## How to Work on This Project

### 1. Read Before Writing

Before implementing any tool:
1. Read `ROADMAP.md` for the phase and task you're working on
2. Read `ai-coreutils-roadmap.md` for the full XML schema spec of that tool
3. Read existing tools in `tools/` to match patterns
4. Read `internal/` packages for shared utilities

### 2. Implementation Order

Follow the roadmap phases strictly. Do NOT skip ahead:

```
Phase 0: Foundation → ls
Phase 1: cat → grep → find → stat → wc → diff
Phase 2: file → head/tail → du/df → path utils → pwd → sort/uniq → cut/tr → env → system → ps → checksums
Phase 3: Performance, cross-platform, docs, CI
```

Each tool must pass all tests before moving to the next.

### 3. File Structure

```
aict/
├── cmd/aict/main.go              # Subcommand dispatch, global flags
├── internal/
│   ├── xml/encoder.go            # Shared XML/JSON/plain output
│   ├── detect/language.go        # Extension → language map
│   ├── detect/mime.go            # Magic bytes MIME detection
│   ├── path/resolve.go           # Absolute path resolution
│   ├── format/size.go            # Bytes → human-readable
│   └── meta/timestamp.go         # Unix time + ago_s helpers
└── tools/
    ├── ls/ls.go
    ├── cat/cat.go
    ├── grep/grep.go
    ├── find/find.go
    ├── stat/stat.go
    ├── wc/wc.go
    ├── diff/diff.go
    └── ... (one package per tool)
```

### 4. Tool Implementation Pattern

Every tool follows this structure:

```go
package toolname

import (
    "encoding/xml"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/ermcy/aict/internal/detect"
    "github.com/ermcy/aict/internal/format"
    "github.com/ermcy/aict/internal/meta"
    "github.com/ermcy/aict/internal/path"
    "github.com/ermcy/aict/internal/xmlout"
)

type Config struct {
    // Tool-specific flags
    XML     bool
    JSON    bool
    Plain   bool
    // ... tool flags
}

func Run(args []string) error {
    cfg := parseFlags(args)
    // ... tool logic
    return output(cfg, result)
}

func parseFlags(args []string) Config {
    // Use stdlib flag package
    // Return Config with all flags set
}

func output(cfg Config, result interface{}) error {
    if cfg.JSON {
        return xmlout.WriteJSON(os.Stdout, result)
    }
    if cfg.Plain {
        return writePlain(os.Stdout, result)
    }
    return xmlout.WriteXML(os.Stdout, result, cfg.XML)
}
```

### 5. XML Output Rules

**MANDATORY** — every tool output must follow these rules:

- Root element named after tool: `<ls>`, `<grep>`, `<find>`, etc.
- Root element always has `timestamp` attribute (Unix epoch integer)
- Root element always has all flags/options as attributes
- Root element always has summary counts where applicable
- Every path has both `path` (as given) and `absolute` (resolved) attributes
- All times are Unix epoch integers with companion `_ago_s` attributes
- All sizes are bytes with companion `_human` attributes
- Booleans are `true`/`false` strings, never `1`/`0`
- Errors are `<error code="" msg=""/>` elements, never stderr
- Empty results are valid: `<grep matched_files="0" total_matches="0"/>`
- Binary files never output as CDATA — omit content or use base64
- Language values: lowercase canonical (`go`, `python`, `typescript`, etc.)

### 6. Testing Requirements

Every tool MUST have:

```go
// toolname_test.go
package toolname

import (
    "bytes"
    "encoding/xml"
    "os"
    "path/filepath"
    "testing"
)

func TestToolName_Basic(t *testing.T) {
    // Create temp dir/files
    dir := t.TempDir()
    // ... setup

    // Run tool
    var buf bytes.Buffer
    err := RunWithOutput(dir, &buf, Config{XML: true})
    if err != nil {
        t.Fatal(err)
    }

    // Validate XML is well-formed
    var result ResultType
    if err := xml.Unmarshal(buf.Bytes(), &result); err != nil {
        t.Fatalf("invalid XML: %v\n%s", err, buf.String())
    }

    // Assert expected values
    // ...
}

func TestToolName_Error(t *testing.T) {
    // Test error handling: missing files, permission denied, etc.
}

func TestToolName_Empty(t *testing.T) {
    // Test empty results produce valid XML
}

func TestToolName_EdgeCases(t *testing.T) {
    // Tool-specific edge cases
}
```

**Test checklist per tool**:
- [ ] Basic functionality with valid input
- [ ] Error handling (missing files, permission denied)
- [ ] Empty results (valid XML, zero counts)
- [ ] Edge cases (empty files, symlinks, binary files, large files)
- [ ] Flag combinations
- [ ] Plain text passthrough mode
- [ ] JSON mode

### 7. Code Style

- Standard `gofmt` formatting
- No comments unless explaining non-obvious algorithm (Myers diff)
- Error messages: lowercase, no punctuation, match Go stdlib style
- Function names: `Run`, `parseFlags`, `output`, `searchFile`, etc.
- Struct fields: exported for XML marshaling, unexported for internal use
- Use `t.TempDir()` for test fixtures, never hardcoded paths
- Use `bytes.Buffer` for output capture in tests

### 8. Dependencies

**ALLOWED** — Go standard library only:
- `os`, `io/fs`, `io`, `bufio`
- `path/filepath`, `path`
- `regexp`, `strings`, `strconv`, `fmt`
- `encoding/xml`, `encoding/json`
- `net/http` (MIME detection only)
- `syscall`, `runtime`, `os/user`
- `crypto/md5`, `crypto/sha256`, `crypto/sha1`
- `time`, `sort`, `slices`, `unicode`, `unicode/utf8`
- `flag`

**FORBIDDEN** — no external dependencies:
- No `github.com/...` imports
- No `go get` for third-party packages
- No cgo unless explicitly approved (Tree-sitter in Phase 4 only)

### 9. Build & Test Commands

```bash
# Build
go build -o aict ./cmd/aict

# Test all
go test ./...

# Test specific tool
go test ./tools/ls/

# Test with coverage
go test -cover ./...

# Vet
go vet ./...

# Format
gofmt -w .

# Benchmark (Phase 3)
go test -bench=. ./tools/grep/
```

### 10. Commit Messages

Use Conventional Commits format:

```
feat(ls): implement directory listing with XML output
feat(grep): add recursive search with context lines
fix(stat): handle symlinks correctly with Lstat
test(wc): add edge case tests for empty files
docs: add XML schema reference for all Tier 1 tools
perf(grep): use buffered I/O for large file scanning
```

### 11. What NOT to Do

- **Do NOT** implement Tier 3 tools (`cp`, `mv`, `rm`, `mkdir`, etc.) — they're write operations, out of scope
- **Do NOT** add external dependencies — zero non-stdlib is a hard constraint
- **Do NOT** output to stderr — all errors are structured XML in stdout
- **Do NOT** use relative paths in output — always absolute
- **Do NOT** use locale-dependent strings — epoch integers and bytes only
- **Do NOT** skip tests — every tool must have full test coverage
- **Do NOT** implement tools out of order — follow the roadmap phases
- **Do NOT** output binary content as CDATA — omit or use base64
- **Do NOT** panic on errors — return structured `<error>` elements
- **Do NOT** buffer entire results for huge directories — use streaming (Phase 4)

### 12. Verification Checklist

Before marking any tool as complete:

```bash
# 1. Builds without errors
go build -o aict ./cmd/aict

# 2. All tests pass
go test ./tools/toolname/

# 3. No vet warnings
go vet ./tools/toolname/

# 4. XML output is valid
AICT_XML=1 ./aict toolname [args] | xmllint --noout -

# 5. JSON output is valid
./aict toolname [args] --json | python -m json.tool > /dev/null

# 6. Plain text mode works
./aict toolname [args] --plain

# 7. Error handling works
./aict toolname /nonexistent --xml
# → Should output <error> element, not panic, not write to stderr

# 8. Empty results work
./aict grep "neverexists" . --xml
# → Should output valid XML with zero counts

# 9. Output matches spec in ai-coreutils-roadmap.md
# → Compare XML structure attribute by attribute
```

### 13. Common Pitfalls

| Pitfall | How to avoid |
|---------|-------------|
| Forgetting `absolute` path attribute | Always call `path.Resolve()` on every path |
| Using `os.Stat` instead of `os.Lstat` | `Lstat` for symlinks, `Stat` only with `-L` flag |
| Locale-dependent time formatting | Always use `time.Now().Unix()` |
| Human-readable sizes instead of bytes | Always store bytes, add `_human` as companion |
| Buffering entire XML in memory | Use `xml.NewEncoder(writer)` for streaming |
| Not handling binary files | Check MIME type, skip CDATA for binary |
| Forgetting `_ago_s` companion attributes | Use `meta.AgoSeconds(unixTime)` helper |
| Panic on missing files | Return `<error>` element, exit 0 |
| Relative paths in output | Always resolve to absolute before output |
| Not testing empty results | Write explicit `TestTool_Empty` test |

---

## Quick Reference: Tool-to-Package Map

| Tool | Package | Key stdlib | Enrichment |
|------|---------|-----------|------------|
| `ls` | `tools/ls` | `os`, `io/fs` | MIME, language, binary flag |
| `cat` | `tools/cat` | `os`, `bufio` | Encoding, language, line count |
| `grep` | `tools/grep` | `regexp`, `bufio` | Col, offset, context, language |
| `find` | `tools/find` | `io/fs`, `path/filepath` | Depth, language, MIME |
| `stat` | `tools/stat` | `os`, `syscall` | `_ago_s` times, language |
| `wc` | `tools/wc` | `bufio`, `unicode` | Language per file |
| `diff` | `tools/diff` | `strings` (Myers) | Change type, line numbers |
| `file` | `tools/file` | `net/http` | Category, charset |
| `head/tail` | `tools/head`, `tools/tail` | `bufio` | Truncated flag, totals |
| `du/df` | `tools/du`, `tools/df` | `io/fs`, `syscall` | Human sizes, inodes |
| `path utils` | `tools/realpath`, etc. | `path/filepath` | Exists, type, stem, ext |
| `pwd` | `tools/pwd` | `os` | Relative to home |
| `sort/uniq` | `tools/sort`, `tools/uniq` | `sort`, `bufio` | Rank, count |
| `cut/tr` | `tools/cut`, `tools/tr` | `strings`, `unicode` | Original + extracted |
| `env` | `tools/env` | `os` | Secret redaction, PATH parsing |
| `system` | `tools/system` | `os/user`, `runtime` | Groups, distro, runtime |
| `ps` | `tools/ps` | `os` (procfs) | State description, exe path |
| `checksums` | `tools/checksums` | `crypto/*` | Multiple hashes at once |

---

## When in Doubt

1. Check `ai-coreutils-roadmap.md` for the XML schema spec
2. Check `ROADMAP.md` for the implementation phase and acceptance criteria
3. Check existing tools in `tools/` for patterns
4. If still unsure, ask before implementing
