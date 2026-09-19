# Usage

## Basic Commands

Every `aict` tool is invoked as a subcommand:

```bash
aict <tool> [flags] [arguments]
```

### Examples

```bash
# List directory contents
aict ls src/

# Search for a pattern
aict grep "func main" . -r

# Read a file
aict cat main.go

# Find files by name
aict find . -name "*.go"

# Get file metadata
aict stat main.go

# Compare two files
aict diff old.go new.go

# Count lines, words, bytes
aict wc main.go

# Text processing
aict sed main.go -e 's/foo/bar/g'
aict awk main.go -f '{print $1}'
aict jq data.json --path '.key'
aict tar archive.tar.gz -t

# Start MCP server (stdio transport)
aict mcp
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--xml` | Force XML output |
| `--json` | Force JSON output |
| `--plain` | Plain text output (GNU-compatible) |
| `--help` | Show help for any tool |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AICT_XML=1` | Force XML output over an earlier `--json`/`--plain` (XML is the default) |
| `AICT_NOCOMPACT=1` | Verbose attribute names and `true`/`false` booleans |

## Help

```bash
# List all available tools
aict help

# Get help for a specific tool
aict grep --help
```
