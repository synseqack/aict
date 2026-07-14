<div align="center">

<img src="./public/logo.svg" width="380px" alt="aict" />

**Unix coreutils with XML/JSON output — built for AI agents, not humans.**

[![CI](https://img.shields.io/github/actions/workflow/status/synseqack/aict/ci.yml?branch=master&label=CI&style=flat-square)](https://github.com/synseqack/aict/actions)
[![Go 1.25](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/synseqack/aict?style=flat-square)](https://github.com/synseqack/aict/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/synseqack/aict?style=flat-square)](https://goreportcard.com/report/github.com/synseqack/aict)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](LICENSE)
[![Stars](https://img.shields.io/github/stars/synseqack/aict?style=flat-square)](https://github.com/synseqack/aict/stargazers)

[Install](#install) · [Quick start](#quick-start) · [All tools](#tools) · [MCP server](#mcp-server) · [Claude Code](#claude-code-integration) · [Token cost](#token-cost) · [Benchmarks](#benchmarks) · [Contributing](CONTRIBUTING.md)

</div>

---

## The problem

AI agents run `ls`, `grep`, and `cat` and get back **human-readable plaintext**. Then they spend tokens parsing column positions, guessing field widths, and handling inconsistent formats. This is fragile and wasteful.

```
-rw-r--r-- 1 user staff  2048 Apr  6 10:00 main.go        ← which column is size?
-rw-r--r-- 1 user staff  1024 Apr  6 10:00 utils.go       ← what's the language?
drwxr-xr-x 5 user staff   160 Apr  6 10:00 internal       ← is this a directory?
```

## The solution

`aict` reimplements 33 Unix tools with **structured output** the agent can read directly — no parsing required.

```xml
$ aict ls src/
<ls timestamp="1746123456" total_entries="3">
  <file name="main.go" path="src/main.go" absolute="/project/src/main.go"
        size_bytes="2048" size_human="2.0K" language="go" mime="text/x-go"
        binary="false" executable="false" modified="1746120000" modified_ago_s="3456"/>
  <file name="utils.go" path="src/utils.go" absolute="/project/src/utils.go"
        size_bytes="1024" size_human="1.0K" language="go" mime="text/x-go"
        binary="false" executable="false" modified="1746120000" modified_ago_s="3456"/>
  <directory name="internal" path="src/internal" modified="1746120000"/>
</ls>
```

Every field is labeled. Paths are always absolute. Timestamps are Unix integers. Language and MIME type are detected automatically — zero parsing needed.

---

## Install

### Homebrew (macOS)

```bash
brew tap synseqack/aict
brew install aict
```

This installs `aict` plus shell completions for bash and zsh. The MCP server is built in: `aict mcp`.

### Go Install

```
go install github.com/synseqack/aict@latest
```

### Build from Source

```
git clone https://github.com/synseqack/aict
cd aict
go build -o aict .
```

> **Verify install:** `aict --help` should list all available tools.

---

## Quick start

```sh
# Default: XML output (best for AI agents)
aict ls src/
aict grep "func" . -r
aict cat main.go
aict diff old.go new.go

# JSON output
aict ls src/ --json

# Plain text (same as the original Unix tools)
aict ls src/ --plain

# Enable XML globally for all aict calls
export AICT_XML=1
```

---

## Tools

33 tools across 6 categories. Every tool supports `--xml` (default), `--json`, and `--plain`.

| Category | Tools |
|----------|-------|
| **File inspection** | `cat` `head` `tail` `file` `stat` `wc` |
| **Search & compare** | `ls` `find` `grep` `diff` |
| **Path utilities** | `realpath` `basename` `dirname` `pwd` |
| **Text processing** | `sort` `uniq` `cut` `tr` `sed` `awk` |
| **Data & archives** | `jq` `tar` |
| **System & environment** | `env` `system` `ps` `df` `du` `checksums` `md5sum` `sha1sum` `sha256sum` |

Additional: `git` (status, diff, log, ls-files, blame) · `completions` (bash/zsh/fish) · `doctor` (self-diagnostic)

---

## Output format

All tools follow the same conventions:

| Field | Convention |
|-------|-----------|
| Paths | Always absolute (`absolute` attr) |
| Timestamps | Unix epoch integers + `_ago_s` companion |
| Sizes | Bytes (`size_bytes`) + human-readable (`size_human`) |
| Booleans | `"true"` / `"false"` strings |
| Errors | `<error code="" msg=""/>` elements — never stderr |
| Empty results | Valid XML with zero counts, never an error |

---

## MCP server

`aict mcp` exposes all tools as callable MCP functions via stdio transport. AI assistants call them natively — no shell wrapping needed. The MCP server is a subcommand of the main binary.

**Configure Claude Desktop** (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "aict": {
      "command": "aict",
      "args": ["mcp"]
    }
  }
}
```

If `aict` is not in PATH, use its full path:

```json
{
  "mcpServers": {
    "aict": {
      "command": "/usr/local/bin/aict",
      "args": ["mcp"]
    }
  }
}
```

---

## Claude Code integration

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "aict": {
      "command": "aict",
      "args": ["mcp"]
    }
  }
}
```

Once connected, Claude Code can call `ls`, `grep`, `diff`, and all other tools as native functions with typed arguments and structured JSON results.

---

## Token cost

Honest numbers first: **aict output costs 1.1–7.8× more tokens per task than terse GNU output** (measured with tiktoken `o200k_base`). What those tokens buy: fewer round-trips and zero parsing ambiguity. Where plaintext needs 2–4 chained calls (`ls` then `file` for languages, `find` then `stat` per hit), aict answers in one.

| Task | GNU | aict | Tokens (GNU → aict) |
|------|-----|------|---------------------|
| List dir with size/type/language | 2 calls | 1 call | 246 → 820 · 3.3× |
| Read file + line count + type | 3 calls | 1 call | 173 → 274 · 1.6× |
| Find `.go` files with size/mtime | 4 calls | 1 call | 47 → 367 · 7.8× |
| Grep with file/line/context | 1 call | 1 call | 141 → 326 · 2.3× |
| Diff with change types | 1 call | 1 call | 167 → 192 · 1.2× |

Every extra plaintext call is a full agent turn — model inference, tool-call overhead, and intermediate output all land in the context window anyway, none of which the token counts above include. And in this very benchmark, `file(1)` misidentified a Go source file as "C source"; aict labeled it `go`.

Use `--plain` when you only need raw content. See [`benchmarks/TOKENS.md`](benchmarks/TOKENS.md) for methodology; reproduce with `go run ./cmd/tokenbench`.

---

## Benchmarks

aict trades some speed for semantic richness (language detection, MIME typing, absolute paths). The overhead is intentional. Startup cost is ~3.6 ms per invocation.

| Tool | GNU | `--plain` | `--xml` | Notes |
|------|-----|-----------|---------|-------|
| `diff` (1000 lines) | 0.9 ms | 1.9 ms · 2.1× | 2.1 ms · 2.4× | ✅ Myers O(ND) |
| `wc` (100k lines) | 6.1 ms | 16 ms · 2.6× | 17 ms · 2.7× | ✅ |
| `awk` (10k lines) | 4.1 ms | 12 ms · 2.9× | 11 ms · 2.6× | ✅ |
| `sed` (10k lines) | 3.3 ms | 14 ms · 4.2× | 16 ms · 4.9× | ✅ |
| `find` (deep tree) | 1.9 ms | 13 ms · 6.8× | 15 ms · 8.0× | ✅ |
| `ls` (1000 files) | 4.0 ms | 51 ms · 12.9× | 70 ms · 17.7× | MIME+lang detection per file |
| `cat` (100k lines) | 1.4 ms | 24 ms · 16.4× | 31 ms · 21.6× | line-by-line scan + encoding detect |
| `grep` (100k lines) | 1.3 ms | 119 ms · 88× | 130 ms · 96× | Go regexp vs GNU SIMD |

Medians from 5 runs on Linux/amd64. See [`benchmarks/`](benchmarks/) for methodology and `make bench` to reproduce.

Use `--plain` to skip enrichment when you only need raw content.

---

## FAQ

**Why XML and not JSON by default?**

XML attributes are denser in a context window. `<file size="1024" lang="go"/>` is shorter than `{"size":1024,"lang":"go"}`. Use `--json` if you prefer JSON — the structure is identical.

**Why not pipe GNU tools to `jq`?**

`ls`, `cat`, `stat`, `find`, `diff`, and `wc` don't output JSON. `jq` can't help with them. aict provides structured output for the entire toolchain, not just grep. (aict also ships its own `jq` for querying JSON files with path expressions.)

**How does this compare to ripgrep?**

ripgrep is much faster for pure search. aict grep adds language detection, MIME type, and a consistent output format shared with every other tool. Use ripgrep for speed-critical search; use aict when the agent needs structured context.

**How does this compare to eza / lsd?**

eza and lsd are better `ls` for humans — great colors and formatting. aict outputs data structures, not formatted tables. They're solving different problems.

**Does it work on Windows?**

`ls`, `cat`, `stat`, `wc`, `find`, `diff`, `grep`, `head`, `tail`, `sort`, `uniq`, `cut`, `tr`, `sed`, `awk`, `jq`, `tar`, `checksums`, and path utilities work on Windows. `ps`, `df`, and `system` are Linux/macOS only.

**Is this safe to run in a sandboxed environment?**

Yes. aict is strictly read-only. No network requests (MIME detection uses the Go stdlib, not HTTP). No telemetry. No data collection. It only reads paths you explicitly pass to it.

**How many dependencies does it have?**

One: the official [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk), used only by the `aict mcp` subcommand. All 33 tools and every internal package are pure Go standard library — enforced as a hard constraint in [AGENTS.md](AGENTS.md).

---

## Contributing

Bug reports, feature requests, and PRs are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines, code style, and the tool implementation pattern.

Issues tagged [`good first issue`](https://github.com/synseqack/aict/issues?q=label%3A%22good+first+issue%22) are a good place to start.

---

## License

[MIT](LICENSE) — built entirely by AI tools, for AI tools.