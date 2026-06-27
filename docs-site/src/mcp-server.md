# MCP Server

`aict` includes a built-in MCP (Model Context Protocol) server so AI assistants can call every tool directly — no shell spawning, no output parsing. The MCP server is a subcommand of the same binary: `aict mcp`.

## Configure Claude Desktop

Add to `~/.config/claude/claude_desktop_config.json`:

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

## Configure Cursor

Add to `.cursor/mcp.json` in your project root:

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

## Available Tools

Once connected, every `aict` tool becomes a typed, callable function:

| Tool | Description |
|------|-------------|
| `ls` | Directory listing |
| `grep` | Pattern search |
| `cat` | File read |
| `find` | Filesystem search |
| `stat` | File metadata |
| `wc` | Line/word/byte count |
| `diff` | File comparison |
| `git status` | Git status |
| `git diff` | Git diff |
| `git log` | Git log |
| `git ls-files` | Git tracked files |
| `git blame` | Git blame |
| `checksums` | Hash computation |
| `ps` | Process list |
| `env` | Environment variables |
| `system` | System info |
| `doctor` | Self-diagnostic |

## How It Works

1. The MCP server starts and registers all tool specs
2. The AI client (Claude, Cursor) discovers available tools
3. When the model decides to call a tool, it sends a structured request
4. The server runs the tool, captures structured JSON output, and returns it
5. The model receives typed data — not raw terminal output

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Server won't start | Ensure `aict` is in your PATH |
| Tools not showing | Restart your AI client after config change |
| Permission errors | Check file permissions on target directories |
