# aict — AI Coreutils

A single Go binary reimagining ~22 Unix CLI tools (`ls`, `grep`, `cat`, `find`, `diff`, etc.) with **structured XML output** — built for AI coding agents that parse tool output instead of humans.

## Why

AI agents waste tokens parsing human-readable `ls`, `grep`, and `cat` output. XML output carries **3× semantic density** with zero ambiguity. Every file listing, search result, and diff comes enriched with language detection, MIME types, absolute paths, and structured errors.

```xml
<ls timestamp="1712345678" files="3" directories="1">
  <file path="main.go" absolute="/home/user/project/main.go"
        size_bytes="1024" size_human="1.0 KiB"
        modified="1712345600" modified_ago_s="78"
        language="go" mime="text/x-go" binary="false" executable="false"/>
</ls>
```

## Quick Start

```bash
# Install
go install github.com/ermcy/aict@latest

# XML output (default for AI agents)
AICT_XML=1 aict ls src/

# Plain text (compatibility mode)
aict ls src/ --plain

# JSON output
aict ls src/ --json
```

## Tools

### Tier 1 — Core Reads

| Tool | Replaces | What it does |
|------|----------|-------------|
| `ls` | `ls` | Directory listing with MIME, language, binary detection |
| `cat` | `cat` | File read with encoding detection, line counts |
| `grep` | `grep` | Content search with context lines, column offsets |
| `find` | `find` | Filesystem search with depth, mtime, size filters |
| `stat` | `stat` | File metadata with `_ago_s` timestamps |
| `wc` | `wc` | Line, word, char, byte counts per file |
| `diff` | `diff` | Myers diff with unified format, change types |

### Tier 2 — Contextual Enrichment

| Tool | Replaces | What it does |
|------|----------|-------------|
| `file` | `file` | Magic bytes + extension type detection |
| `head` / `tail` | `head` / `tail` | Partial read with truncation flags |
| `du` / `df` | `du` / `df` | Disk usage and filesystem stats |
| `realpath` | `realpath` | Absolute path resolution with existence check |
| `basename` | `basename` | Path stem and extraction |
| `dirname` | `dirname` | Parent directory extraction |
| `pwd` | `pwd` | Working directory with home-relative path |
| `sort` / `uniq` | `sort` / `uniq` | Line sorting and deduplication with counts |
| `cut` / `tr` | `cut` / `tr` | Field extraction and character transliteration |
| `env` | `env` | Environment snapshot with secret redaction |
| `system` | `id` / `whoami` / `uname` | User, OS, and runtime info combined |
| `ps` | `ps` | Process listing with state descriptions |
| `checksums` | `md5sum` / `sha256sum` | Multiple hash algorithms at once |

## Output Modes

| Mode | Flag | Use case |
|------|------|----------|
| **XML** | `--xml` or `AICT_XML=1` | AI agent consumption (default) |
| **JSON** | `--json` | Programmatic integration |
| **Plain** | `--plain` | Human readability, compatibility |

## Design Principles

- **Zero non-stdlib dependencies** — single binary, no `go get`
- **All paths absolute** — `path` (as given) + `absolute` (resolved) in every output
- **All timestamps Unix epoch** — integers with `_ago_s` companions, no locale strings
- **All sizes in bytes** — with `_human` companions (KiB, MiB, GiB)
- **Structured errors** — `<error code="" msg=""/>` elements, never stderr
- **Empty results are valid** — `<grep matched_files="0" total_matches="0"/>`
- **Binary-safe** — no CDATA for binary content, omit or use base64

## XML Output Rules

Every tool follows these rules:

- Root element named after tool: `<ls>`, `<grep>`, `<find>`, etc.
- Root has `timestamp` attribute (Unix epoch integer)
- Root has all flags/options as attributes
- Root has summary counts where applicable
- Language values: lowercase canonical (`go`, `python`, `typescript`)
- Booleans: `true`/`false` strings, never `1`/`0`

## Comparison with GNU Coreutils

| Feature | GNU coreutils | aict |
|---------|--------------|------|
| Output format | Plaintext, locale-dependent | XML/JSON/plain, machine-parseable |
| Paths | Relative or absolute | Always both |
| Timestamps | Locale strings | Unix epoch + ago seconds |
| Sizes | Locale-dependent | Bytes + human-readable |
| Errors | stderr, exit codes | Structured XML in stdout |
| Dependencies | C library, gettext | Go stdlib only |
| Binary size | ~5MB (multiple binaries) | ~3MB (single binary) |
| AI-friendly | No | Yes |

## Roadmap

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| 0 — Foundation | Week 1 | Project scaffold + `ls` working |
| 1 — Core Reads | Week 2-3 | All 7 Tier 1 tools working |
| 2 — Enrichment | Week 4-5 | All 15 Tier 2 tools working |
| 3 — Polish | Week 6 | Benchmarks, cross-platform, docs, CI |
| 4 — Future | Post-MVP | JSON mode, git tools, MCP server |

## Development

```bash
# Build
go build -o aict ./cmd/aict

# Test all
go test ./...

# Test specific tool
go test ./tools/ls/

# Vet
go vet ./...

# Format
gofmt -w .
```

## License

MIT
