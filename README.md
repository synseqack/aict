<div align="center">

<img src="./public/logo.svg" width="380px" alt="aict" />

**Unix coreutils with XML/JSON output — built for AI agents, not humans.**

[![CI](https://img.shields.io/github/actions/workflow/status/synseqack/aict/ci.yml?branch=main&label=CI&style=flat-square)](https://github.com/synseqack/aict/actions)
[![Go 1.25](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/synseqack/aict?style=flat-square)](https://github.com/synseqack/aict/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/synseqack/aict?style=flat-square)](https://goreportcard.com/report/github.com/synseqack/aict)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](LICENSE)
[![Stars](https://img.shields.io/github/stars/synseqack/aict?style=flat-square)](https://github.com/synseqack/aict/stargazers)

[Install](#install) · [Quick start](#quick-start) · [All tools](#tools) · [MCP server](#mcp-server) · [Claude Code](#claude-code-integration) · [Benchmarks](#benchmarks) · [Contributing](CONTRIBUTING.md)

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

`aict` reimplements 22 Unix tools with **structured output** the agent can read directly — no parsing required.

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

This installs both `aict` and `aict-mcp` binaries, plus shell completions for bash and zsh.

### Go Install

```
go install github.com/synseqack/aict@latest
go install github.com/synseqack/aict/cmd/mcp@latest
```

### Build from Source

```
git clone https://github.com/synseqack/aict
cd aict
go build -o aict .
go build -o aict-mcp ./cmd/mcp
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

22 tools across 5 categories. Every tool supports `--xml` (default), `--json`, and `--plain`.

| Category | Tools |
|----------|-------|
| **File inspection** | `cat` `head` `tail` `file` `stat` `wc` |
| **Search & compare** | `ls` `find` `grep` `diff` |
| **Path utilities** | `realpath` `basename` `dirname` `pwd` |
| **Text processing** | `sort` `uniq` `cut` `tr` |
| **System & environment** | `env` `system` `ps` `df` `du` `checksums` |

Additional: `git` (status, diff, log, ls-files, blame) · `doctor` (self-diagnostic)

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

`aict-mcp` exposes all 22 tools as callable MCP functions. AI assistants call them natively — no shell wrapping needed.

**Build:**

```sh
go build -o aict-mcp ./cmd/mcp
```

**Configure Claude Desktop** (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "aict": {
      "command": "aict-mcp",
      "args": []
    }
  }
}
```

If `aict` is not in PATH, set the binary location explicitly:

```json
{
  "mcpServers": {
    "aict": {
      "command": "aict-mcp",
      "env": {
        "AICT_BINARY": "/full/path/to/aict"
      }
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
      "command": "aict-mcp",
      "args": []
    }
  }
}
```

Once connected, Claude Code can call `ls`, `grep`, `diff`, and all other tools as native functions with typed arguments and structured JSON results.

---

## Benchmarks

aict trades some speed for semantic richness (language detection, MIME typing, absolute paths). The overhead is intentional.

| Tool | GNU | aict | Ratio | Notes |
|------|-----|------|-------|-------|
| `ls` (1000 files) | ~2ms | ~15ms | 7x | ✅ within target |
| `find` (deep tree) | ~2ms | ~9ms | 5x | ✅ within target |
| `diff` (1000 lines) | ~1ms | ~10ms | 10x | ✅ within target |
| `grep` (100k lines) | ~1ms | ~100ms | 100x | language detection per file |
| `cat` (100k lines) | ~1ms | ~23ms | 17x | encoding + MIME detection |

Use `--plain` to skip enrichment when you only need raw content.

---

## FAQ

**Why XML and not JSON by default?**

XML attributes are denser in a context window. `<file size="1024" lang="go"/>` is shorter than `{"size":1024,"lang":"go"}`. Use `--json` if you prefer JSON — the structure is identical.

**Why not pipe GNU tools to `jq`?**

`ls`, `cat`, `stat`, `find`, `diff`, and `wc` don't output JSON. `jq` can't help with them. aict provides structured output for the entire toolchain, not just grep.

**How does this compare to ripgrep?**

ripgrep is much faster for pure search. aict grep adds language detection, MIME type, and a consistent output format shared with every other tool. Use ripgrep for speed-critical search; use aict when the agent needs structured context.

**How does this compare to eza / lsd?**

eza and lsd are better `ls` for humans — great colors and formatting. aict outputs data structures, not formatted tables. They're solving different problems.

**Does it work on Windows?**

`ls`, `cat`, `stat`, `wc`, `find`, `diff`, `grep`, `head`, `tail`, `sort`, `uniq`, `cut`, `tr`, `checksums`, and path utilities work on Windows. `ps`, `df`, and `system` are Linux/macOS only.

**Is this safe to run in a sandboxed environment?**

Yes. aict is strictly read-only. No network requests (MIME detection uses the Go stdlib, not HTTP). No telemetry. No data collection. It only reads paths you explicitly pass to it.

---

## Contributing

Bug reports, feature requests, and PRs are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines, code style, and the tool implementation pattern.

Issues tagged [`good first issue`](https://github.com/synseqack/aict/issues?q=label%3A%22good+first+issue%22) are a good place to start.

---

## License

[MIT](LICENSE) — built entirely by AI tools, for AI tools.