# Output Modes

Every `aict` tool supports three output modes.

## XML (Default for AI)

XML is the default. `AICT_XML=1` forces it over an earlier `--json` or `--plain`; `--xml` selects it for one call.

```xml
<ls timestamp="1234567890" total_entries="1">
  <file name="main.go" path="main.go" absolute="/project/main.go"
        size_bytes="2048" size_human="2.0 KiB"
        modified="1234567890" modified_ago_s="3600"
        language="go" mime="text/x-go" binary="false"/>
</ls>
```

**Why XML?** Attributes carry metadata without nesting. A `<file size_bytes="2048" language="go"/>` is 40 chars; the JSON equivalent is 60+. AI context windows are token-limited — denser encoding means more results per call.

## JSON

Pass `--json` for JSON output. The schema mirrors the XML structure.

```json
{
  "timestamp": 1234567890,
  "total_entries": 1,
  "files": [
    {
      "name": "main.go",
      "path": "main.go",
      "absolute": "/project/main.go",
      "size_bytes": 2048,
      "size_human": "2.0 KiB",
      "modified": 1234567890,
      "modified_ago_s": 3600,
      "language": "go",
      "mime": "text/x-go",
      "binary": false
    }
  ]
}
```

## Plain Text

Pass `--plain` for GNU-compatible plain text output. This skips enrichment and outputs text similar to the original Unix tool.

```bash
$ aict ls src/ --plain
main.go
utils.go
internal/
```

Use `--plain` when:
- You only need content, not metadata
- You want maximum performance (skips language/MIME detection)
- You need compatibility with existing scripts

## Error Output

Errors are structured XML in stdout, never stderr:

```xml
<ls timestamp="1234567890" total_entries="0">
  <error code="2" msg="no such file or directory" path="/nonexistent"/>
</ls>
```

Exit code is always 0 for structured errors. Non-zero exit codes indicate fatal failures.
