# Integration Guide: Using aict with AI Coding Agents

This guide explains how to integrate `aict` into your AI coding workflow for maximum token efficiency.

## Why aict for AI Agents

AI coding agents (Claude, Cursor, GitHub Copilot, etc.) call CLI tools constantly. Standard tools output human-readable text that the agent must parse:

```
# Standard ls output - agent must guess columns
-rw-r--r--  1 user  staff  2048 Apr  6 12:00 main.go
```

With `aict`, the agent receives structured data with zero ambiguity:

```xml
<file name="main.go" path="main.go" absolute="/project/main.go"
      size_bytes="2048" modified="1712404800" language="go" mime="text/x-go"/>
```

**Token savings**: ~40% fewer tokens for equivalent information density.

## Setup

### Shell Configuration

Add to your `.bashrc` or `.zshrc`:

```bash
# Enable XML output for all aict commands
export AICT_XML=1
```

### MCP Server (Recommended)

The MCP server exposes every `aict` tool as a callable function to AI assistants.

1. Build the MCP server:
    ```bash
    go build -o aict-mcp ./cmd/mcp
    ```

2. Configure your AI client (see README.md for Claude/Cursor configs)

3. The agent can now call tools directly without shell spawning

## Output Modes

### XML (Default for AI)

```bash
AICT_XML=1 aict ls src/
```

Best for: AI agents, structured parsing, maximum metadata

### JSON

```bash
aict ls src/ --json
```

Best for: Programmatic consumption, JavaScript/Python scripts

### Plain Text

```bash
aict ls src/ --plain
```

Best for: Human reading, fallback compatibility, performance

## Common Patterns

### Search a Codebase

```bash
# Find all function definitions in Go files
aict grep "func " . -r --include "*.go"
```

The agent receives: matched files, line numbers, byte offsets, context lines, and language metadata — all in one response.

### Understand a Directory

```bash
aict ls src/
```

The agent receives: file sizes, modification times, languages, MIME types, binary flags, and executable status — no guessing.

### Compare Files

```bash
aict diff old.go new.go
```

The agent receives: structured hunks with added/removed/context lines, line numbers, and change counts.

### Read a File

```bash
aict cat main.go
```

The agent receives: content with encoding detection, language identification, line count, and MIME type.

## Error Handling

All errors are structured XML in stdout, never stderr:

```xml
<ls timestamp="1234567890" total_entries="0">
  <error code="2" msg="no such file or directory" path="/nonexistent"/>
</ls>
```

The agent can parse `<error>` elements programmatically without regex.

## Performance Tips

1. **Use `--plain` when you only need content** — skips language/MIME detection
2. **Use `--include` filters in grep** — reduces files scanned
3. **Use `--maxdepth` in find** — limits directory traversal
4. **Use `aict grep` over `rg` when you need structured results** — `rg` is faster but returns plain text

## Shell Completion

```bash
# Bash
source completions/aict.bash

# Zsh
source completions/aict.zsh
```