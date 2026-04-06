# Migration Guide: GNU Coreutils to aict

This guide maps common GNU coreutils commands to their `aict` equivalents.

## Quick Reference

| GNU Command | aict Equivalent | Notes |
|-------------|-----------------|-------|
| `ls -la src/` | `aict ls src/` | XML by default, add `--plain` for ls-like output |
| `cat file.go` | `aict cat file.go` | Add `--plain` for raw content |
| `grep -rn "func" .` | `aict grep "func" . -r` | Returns structured matches with context |
| `find . -name "*.go"` | `aict find . -name "*.go"` | Adds language, MIME, size metadata |
| `stat file.go` | `aict stat file.go` | Includes `_ago_s` time companions |
| `wc -l file.go` | `aict wc file.go` | Returns lines, words, chars, bytes |
| `diff file1 file2` | `aict diff file1 file2` | Structured hunks with line numbers |
| `file file.go` | `aict file file.go` | Adds category, charset, language |
| `head -n 10 file.go` | `aict head file.go -n 10` | Adds truncated flag, totals |
| `tail -n 10 file.go` | `aict tail file.go -n 10` | Same enrichment as head |
| `du -sh src/` | `aict du src/ -s -h` | Structured directory sizes |
| `df -h` | `aict df` | Adds inode counts, use percentage |
| `realpath file.go` | `aict realpath file.go` | Adds exists, type attributes |
| `basename file.go .go` | `aict basename file.go` | Adds stem, extension attributes |
| `dirname src/file.go` | `aict dirname src/file.go` | Returns parent directory |
| `pwd` | `aict pwd` | Adds home-relative path |
| `sort file.txt` | `aict sort file.txt` | Adds line counts, sort metadata |
| `uniq -c file.txt` | `aict uniq file.txt -c` | Adds duplicate counts |
| `cut -d, -f1 file.csv` | `aict cut file.csv -d, -f1` | Adds field metadata |
| `tr 'a-z' 'A-Z'` | `aict tr 'a-z' 'A-Z'` | Reads from stdin or file |
| `env` | `aict env` | Redacts secrets automatically |
| `uname -a` | `aict system` | Combined with user/runtime info |
| `ps aux` | `aict ps` | Adds state descriptions, exe paths |
| `md5sum file.go` | `aict checksums file.go` | Returns MD5, SHA1, SHA256 at once |

## Key Differences

### Output Format

| Aspect | GNU | aict |
|--------|-----|------|
| Default | Plain text | XML |
| Structured data | No | Yes |
| Machine-readable | Requires parsing | Native |
| Human-readable | Yes | Use `--plain` |

### Flags

Most aict tools accept the same short flags as their GNU equivalents (`-l`, `-r`, `-n`, etc.). The difference is in the output format.

### When to Use GNU vs aict

| Scenario | Recommendation |
|----------|---------------|
| Interactive terminal use | GNU coreutils |
| AI agent pipelines | aict |
| Scripting with parsing | aict |
| Quick human inspection | GNU coreutils |
| Structured data consumption | aict |

## Environment Variables

| Variable | Effect |
|----------|--------|
| `AICT_XML=1` | Force XML output for all tools |
| `AICT_JSON=1` | Force JSON output for all tools |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (even if no results) |
| 1 | Error (check `<error>` element in output) |

Unlike GNU tools, aict always exits 0 on success even for empty results. Errors are communicated via structured `<error>` elements in stdout.