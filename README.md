# aict

[![CI](https://github.com/synseqack/aict/actions/workflows/ci.yml/badge.svg)](https://github.com/synseqack/aict/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/synseqack/aict)](https://github.com/synseqack/aict)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/synseqack/aict)](https://github.com/synseqack/aict/releases)

A CLI tool that outputs XML/JSON, built for AI agents to consume directly.

## Disclaimer

This project was built entirely by AI tools, for AI tools. Every line of code, every test, and every decision was made by an AI agent working with another AI agent. There were no human engineers writing code.

## The Problem

When an AI agent runs `ls`, `grep`, or `cat`, it gets human-readable plaintext. The agent must parse column positions, guess which field is the filename, and handle inconsistent formats. This parsing is brittle and breaks easily.

## What You Get

```
$ aict ls src/
<ls timestamp="1234567890" total_entries="3">
  <file name="main.go" path="src/main.go" absolute="/project/src/main.go"
        size_bytes="2048" size_human="2.0 KiB" modified="1234567890" modified_ago_s="3600"
        language="go" mime="text/x-go" binary="false"/>
  <file name="utils.go" path="src/utils.go" absolute="/project/src/utils.go"
        size_bytes="1024" size_human="1.0 KiB" modified="1234567890" modified_ago_s="3600"
        language="go" mime="text/x-go" binary="false"/>
  <directory name="internal" path="src/internal"/>
</ls>
```

Every field is labeled. Every path is absolute. Every timestamp is a Unix epoch integer. The agent knows exactly what it is looking at.

## Install

```bash
go install github.com/synseqack/aict@latest
```

Or build from source:

```bash
git clone https://github.com/synseqack/aict
cd aict
go build -o aict
```

## Usage

```bash
# AI mode (XML output)
AICT_XML=1 aict ls src/

# or with flag
aict ls src/ --xml

# Plain text when you need it
aict ls src/ --plain

# JSON for programmatic use
aict ls src/ --json
```

## Available Tools

| Category | Tools |
|----------|-------|
| **File Inspection** | `cat`, `head`, `tail`, `file`, `stat`, `wc` |
| **Directory & Search** | `ls`, `find`, `grep`, `diff` |
| **Path Utilities** | `realpath`, `basename`, `dirname`, `pwd` |
| **Text Processing** | `sort`, `uniq`, `cut`, `tr` |
| **System & Environment** | `env`, `system`, `ps`, `df`, `du`, `checksums` |

## Output Examples

### XML (Default for AI)

```xml
$ aict grep "func" . -r
<grep pattern="func" recursive="true" searched_files="45" matched_files="12" total_matches="34" timestamp="1234567890">
  <file path="src/main.go" absolute="/project/src/main.go" matches_in_file="3">
    <line number="10" text="func main() {" offset_bytes="234"/>
    <line number="25" text="func init() {" offset_bytes="567"/>
    <line number="42" text="func handleRequest() {" offset_bytes="890"/>
  </file>
</grep>
```

### JSON (for programmatic use)

```json
$ aict ls . --json
{"ls":{"timestamp":1234567890,"total_entries":3,"entries":[{"name":"main.go","size_bytes":2048},...]}}
```

### Plain Text (for humans)

```bash
$ aict ls . --plain
total 12
-rw-r--r-- 1 user staff  2048 Apr  6 10:00 main.go
-rw-r--r-- 1 user staff  1024 Apr  6 10:00 utils.go
drwxr-xr-x 5 user staff   160 Apr  6 10:00 internal
```

## MCP Server

aict can run as an MCP server so AI assistants like ChatGPT can call these tools directly.

```bash
go build -o aict-mcp ./cmd/mcp
```

Configure your AI client to use `aict-mcp` as a command-line MCP server. Each tool becomes a callable function with typed arguments and structured JSON output.

## Docker

```bash
# Build
docker build -t aict .

# Run
docker run --rm -v $(pwd):/data aict ls /data
```

## Cross-Platform

- **Linux**: Full support for all tools
- **macOS**: Full support (ps uses sysctl fallback)
- **Windows**: Subset support (ls, cat, stat, wc, find, diff work)

## Why This Exists

AI coding agents need to read files, search codebases, and compare directories. Standard CLI tools output human-readable text. This gives you the same capabilities, but the output is unambiguous.

## FAQ

**Why not just pipe to jq?**

You can: `aict ls . --json | jq '.total_entries'`

But jq doesn't help with `ls`, `cat`, `find`, or `stat` - those don't output JSON by default. aict gives you structured output natively for every tool.

**Why XML instead of JSON by default?**

XML is more readable for debugging and supports attributes alongside content. Both are supported: use `--json` if you prefer.

**What about eza/ripgrep?**

eza is a prettier `ls`. ripgrep is a faster `grep`. Both still output human-readable text. aict is designed for machine consumption first.

## Design Choices

- Single binary, no dependencies beyond Go standard library
- Every tool works in XML, JSON, or plain text
- All timestamps are Unix epoch integers
- All sizes are in bytes with human-readable companions
- Errors are structured data, never stderr
- Paths are always absolute

## Examples

### Example 1: List directory with full metadata
**User prompt:** "List the files in the src directory with their sizes and types"

**What happens:**
- Calls aict ls with path "src"
- Returns XML with file name, size, language, MIME type, modified timestamp
- Agent can extract specific fields without parsing

### Example 2: Search code and find specific patterns
**User prompt:** "Find all functions named 'handle' in the Go files"

**What happens:**
- Calls aict grep with pattern "func.*handle" and include "*.go"
- Returns XML with file paths, line numbers, matched text
- Each match includes column offset for precise navigation

### Example 3: Get detailed file information
**User prompt:** "What's in main.go? Show me its size and when it was last modified"

**What happens:**
- Calls aict stat with path "main.go"
- Returns XML with permissions, size, owner, timestamps (Unix epoch)
- Agent knows exactly when file was modified without parsing date strings

## Privacy Policy

aict is a read-only CLI tool. It does not collect, store, or transmit any user data. The tool only reads files and directories you explicitly specify and outputs structured data.

For complete privacy information, see: https://github.com/synseqack/aict/blob/master/PRIVACY.md

## License

MIT
