# FAQ

## vs ripgrep (`rg`)?

`ripgrep` is faster and better for interactive search in terminals. `aict grep` is slower but returns structured XML/JSON with line numbers, byte offsets, language metadata, and context — all in one response. For AI agents that need to reason about search results, `aict grep` eliminates a parsing layer entirely.

## vs eza / lsd?

`eza` and `lsd` are beautiful terminal replacements for humans. `aict ls` is for machines — the output is XML with absolute paths, MIME types, language tags, binary flags, and epoch timestamps. There is no colour code to strip, no column alignment to guess.

## Why XML and not JSON by default?

XML is the default because:

- Attributes carry metadata without nesting — a `<file size_bytes="2048" language="go"/>` is 40 chars; the JSON equivalent is 60+ with mandatory quotes and colons.
- AI context windows are token-limited; denser encoding means more results per call.
- Structured errors (`<error code="2" msg="no such file"/>`) compose naturally into the parent element.

Pass `--json` any time you want JSON. The schema is identical.

## Does it work on Windows?

Partially. `ls`, `cat`, `stat`, `wc`, `find`, `diff`, `grep`, `head`, `tail`, `sort`, `uniq`, `cut`, `tr`, `sed`, `awk`, `jq`, `tar`, `checksums`, `realpath`, `basename`, `dirname`, `pwd`, `env` all work. `ps` and `df` are Linux/macOS only (they read `/proc` and use `syscall.Statfs`).

## Can I use it without `AICT_XML=1`?

Yes. Pass `--xml`, `--json`, or `--plain` per invocation. The env var is a convenience for shells configured for AI pipelines.
