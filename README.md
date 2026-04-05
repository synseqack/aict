# aict

Your command line, but built for AI.

## The Problem

Every time an AI agent runs `ls`, `grep`, or `cat`, it spends tokens parsing human-readable output. The agent has to figure out which part is the filename, which is the size, which is the date. This guesswork adds up.

aict gives you the same tools you know, but the output is structured. No parsing. No regex. Just data.

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

Every field is labeled. Every path is absolute. Every timestamp is a Unix epoch integer. The agent knows exactly what it's looking at.

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

- `ls` - Directory listings with language and MIME type detection
- `cat` - File contents with encoding detection  
- `grep` - Search with context lines, recursive support
- `find` - Filesystem search with filters
- `stat` - File metadata with all timestamps
- `wc` - Line, word, and byte counts
- `diff` - Side-by-side comparison

More tools coming.

## Why This Exists

We built AI coding agents that needed to read files, search codebases, and compare directories. Standard CLI tools are designed for humans. Every parsing attempt was brittle.

This gives you the same capabilities, but the output is unambiguous. The agent doesn't guess. It reads.

## Design Choices

- Single binary, no dependencies beyond Go standard library
- Every tool works in XML, JSON, or plain text
- All timestamps are Unix epoch integers
- All sizes are in bytes with human-readable companions
- Errors are structured data, never stderr
- Paths are always absolute

## Something Missing?

This is an open project. If you need a tool added or have a feature request, open an issue.

## License

MIT
