# aict — Coreutils for AI Agents

**Unix coreutils rebuilt for AI agents — structured XML/JSON output, zero parsing, zero ambiguity.**

[![CI](https://github.com/synseqack/aict/actions/workflows/ci.yml/badge.svg)](https://github.com/synseqack/aict/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/synseqack/aict)](https://github.com/synseqack/aict/blob/main/go.mod)
[![Latest release](https://img.shields.io/github/v/release/synseqack/aict)](https://github.com/synseqack/aict/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/synseqack/aict/blob/main/LICENSE)

---

## What is aict?

`aict` is a single Go binary that reimplements 33 Unix CLI tools (`ls`, `grep`, `cat`, `find`, `stat`, `diff`, `sed`, `awk`, `jq`, `tar`, etc.) with structured XML/JSON output designed for AI coding agents.

Every time an AI agent runs `ls`, `grep`, or `cat`, it wastes tokens parsing human-readable output. `aict` gives you the same tools you already know, but the output is structured. No parsing. No regex. Just data.

## What You Get

```xml
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

## Quick Start

```bash
# Install
go install github.com/synseqack/aict@latest

# Use it
aict ls src/
aict grep "func" . -r
aict cat main.go
```

## Key Features

| Feature | Detail |
|---------|--------|
| **Structured output** | XML, JSON, or plain text for every tool |
| **Zero dependencies** | Go standard library only (MCP server excepted) |
| **Single binary** | Drop it anywhere and it works |
| **Enriched metadata** | Language detection, MIME types, epoch timestamps |
| **MCP server** | Built-in: `aict mcp` connects to Claude, Cursor, and other AI assistants |
| **Cross-platform** | Linux, macOS, Windows (partial) |

## Navigate This Site

| Section | What You'll Find |
|---------|-----------------|
| [Installation](./install.md) | Build from source, `go install`, Docker |
| [Usage](./usage.md) | Commands, flags, environment variables |
| [Output Modes](./output-modes.md) | XML, JSON, plain text explained |
| [Tools Reference](./tools-reference.md) | Documentation for all 33 tools |
| [MCP Server](./mcp-server.md) | Connect to Claude, Cursor, etc. |
| [XML Schema Reference](./xml-schema.md) | Complete output schema for every tool |
| [Migration Guide](./migration-guide.md) | GNU coreutils → aict mapping |
| [Integration Guide](./integration-guide.md) | Using aict with AI coding agents |
| [Benchmarks](./benchmarks.md) | Performance vs GNU coreutils |
| [FAQ](./faq.md) | Common questions answered |
| [Contributing](./contributing.md) | How to add tools, code style, testing |

## Why This Exists

We built AI coding agents that needed to read files, search codebases, and compare directories. Standard CLI tools are designed for humans. Every parsing attempt was brittle.

This gives you the same capabilities, but the output is unambiguous. The agent does not guess. It reads.
