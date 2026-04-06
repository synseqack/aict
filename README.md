<h1>
  <img src="./public/logo.svg" alt="aict logo" width="76" style="vertical-align: middle; margin-right: 12px;" />
  aict
</h1>

CLI tools with XML/JSON output for AI agents. Built with AI tools to power AI tools.

[![Go](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/synseqack/aict/actions/workflows/ci.yml/badge.svg)](https://github.com/synseqack/aict/actions)

## The problem

When an AI runs `ls` it gets plaintext like:
```
-rw-r--r-- 1 user staff 1024 Apr  6 10:00 main.go
```
The agent must guess which column is size, date, filename, and so on.

## What you get

```
$ aict ls src/
<ls timestamp="1234567890" total_entries="2">
  <file name="main.go" size_bytes="1024" language="go" mime="text/x-go"/>
  <file name="utils.go" size_bytes="512" language="go" mime="text/x-go"/>
</ls>
```

Every field is labeled. Paths are absolute. Timestamps are Unix integers. Language and MIME type are detected automatically.

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

## Usage

```
# XML (default)
aict ls src/
aict grep "func" . -r

# JSON
aict ls src/ --json

# Plain text
aict ls src/ --plain
```

## Tools

| Category | Tools |
|----------|-------|
| File inspection | cat, head, tail, file, stat, wc |
| Search | ls, find, grep, diff |
| Path | realpath, basename, dirname, pwd |
| Text | sort, uniq, cut, tr |
| System | env, system, ps, df, du, checksums |

## MCP Server

Build: `go build -o aict-mcp ./cmd/mcp`

Add to Claude Desktop (`~/.config/claude/claude_desktop_config.json`):
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

## Claude Code Integration

For Claude Code, add to your `~/.claude.json`:

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

Or set the `AICT_BINARY` environment variable if the binary is not in PATH:
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

## Examples

### Example 1: List files in a directory
**User prompt:** "Show me all files in the src directory"

**Expected behavior:**
- Run `aict ls src/`
- Returns XML with file entries, each containing name, size, language, and MIME type
- Example output:
```xml
<ls timestamp="..." total_entries="2">
  <file name="main.go" size_bytes="1024" language="go" mime="text/x-go"/>
  <file name="utils.go" size_bytes="512" language="go" mime="text/x-go"/>
</ls>
```

### Example 2: Search for a pattern
**User prompt:** "Find all files containing the word 'func' in the project"

**Expected behavior:**
- Run `aict grep "func" . -r`
- Returns XML with matching files, line numbers, and context
- Each match includes the line content and byte offset

### Example 3: Get file metadata
**User prompt:** "Show me detailed info about main.go"

**Expected behavior:**
- Run `aict stat main.go`
- Returns XML with permissions, timestamps, owner, size, language, and MIME type

## Privacy Policy

aict is read-only. It does not collect, store, or transmit any user data. For complete privacy information, see [PRIVACY.md](PRIVACY.md).

### Data Collection
- No data collection
- No network requests (except local MIME type detection)
- No telemetry or analytics
- Only accesses paths explicitly provided

## FAQ

**Why not pipe to jq?**

jq doesn't help with ls, cat, find, stat - they don't output JSON. aict gives structured output for every tool.

**Why XML by default?**

Denser encoding for AI context windows. `<file size="1024"/>` is 22 chars vs 30+ in JSON. Use `--json` if preferred.

**How is this different from ripgrep?**

ripgrep is excellent for searching. aict grep provides similar functionality but adds language detection, MIME type, and integrates with the same output format as other tools. For pure grep performance, use ripgrep with `--json` and parse the output.

**How is this different from eza/lsd?**

eza and lsd are modern ls replacements with better colors and formatting. aict outputs structured data instead of human-readable tables. Use eza for terminal use, aict for AI agent consumption.

**Does it work on Windows?**

Partially. ls, cat, stat, wc, find, diff, grep, head, tail work. ps and df are Linux/macOS only.

**Can I contribute?**

Yes. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Support

For issues and questions: https://github.com/synseqack/aict/issues

## Benchmarks

| Tool | GNU | aict | Notes |
|------|-----|------|-------|
| ls (1000 files) | ~2ms | ~15ms | 7x overhead |
| grep (100k lines) | ~1ms | ~100ms | language detection |
| cat (100k lines) | ~1ms | ~23ms | 17x overhead |

Use `--plain` to skip enrichment when you only need content.

## License

MIT
