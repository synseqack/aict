# AI-Coreutils Implementation Roadmap

> Phased delivery plan for building `aict` — structured CLI tools for AI coding agents.
> Written in Go, zero non-stdlib dependencies, XML-first output.

---

## Phase 0 — Foundation (Week 1)

**Goal**: Project scaffolding, shared infrastructure, first working tool.

### Tasks

- [x] Initialize Go module: `go mod init github.com/synseqack/aict`
- [x] Create directory structure per spec:
  ```
  cmd/aict/main.go
  internal/xml/encoder.go
  internal/detect/language.go
  internal/detect/mime.go
  internal/path/resolve.go
  internal/format/size.go
  internal/meta/timestamp.go
  tools/ls/ls.go
  ```
- [x] Implement `internal/xml/encoder.go`:
  - Pretty vs compact XML output
  - `--xml`, `--json`, `--plain` mode switching
  - `AICT_XML=1` env var detection
  - Structured error element: `<error code="" msg=""/>`
- [x] Implement `internal/detect/language.go`:
  - Extension → language map (~200 entries)
  - Shebang line detection
  - Canonical lowercase names
- [x] Implement `internal/detect/mime.go`:
  - `net/http.DetectContentType` wrapper
  - First 512 bytes sampling
- [x] Implement `internal/path/resolve.go`:
  - `filepath.Abs`, `filepath.EvalSymlinks`
  - Always return absolute paths
- [x] Implement `internal/format/size.go`:
  - Bytes → IEC human-readable (KiB, MiB, GiB)
- [x] Implement `internal/meta/timestamp.go`:
  - `time.Now().Unix()` + `ago_s` helper
- [x] Implement `cmd/aict/main.go`:
  - Subcommand dispatch (`ls`, `cat`, `grep`, etc.)
  - Global flag parsing (`--xml`, `--json`, `--pretty`, `--plain`)
  - Env var `AICT_XML` check
- [x] Implement `tools/ls/ls.go` (Tier 1, first tool):
  - `os.ReadDir` + `fs.FileInfo`
  - XML output per spec: `<ls>`, `<file>`, `<directory>`, `<symlink>`
  - Attributes: `path`, `absolute`, `size_bytes`, `size_human`, `modified`, `modified_ago_s`, `permissions`, `mode`, `owner`, `group`, `mime`, `language`, `binary`, `executable`, `symlink`
  - Flags: `-l`, `-a`, `-A`, `-h`, `-t`, `-r`, `-R`
  - Structured error handling
- [x] Write tests for `ls`:
  - Unit tests for XML output structure
  - Integration tests against real directories
  - Edge cases: empty dirs, symlinks, hidden files, binary files
- [x] `go test ./...` passes
- [x] `go vet ./...` clean
- [x] `go build -o aict ./cmd/aict` produces working binary

### Acceptance Criteria

```bash
# XML mode
AICT_XML=1 ./aict ls src/
# → Valid XML with <ls> root, <file>/<directory> children, all attributes present

# Plain text passthrough
./aict ls src/ --plain
# → Standard ls-like output

# JSON mode
./aict ls src/ --json
# → Valid JSON equivalent

# Error handling
./aict ls /nonexistent --xml
# → <ls><error code="2" msg="no such file or directory"/></ls>
```

---

## Phase 1 — Core Reads (Week 2-3)

**Goal**: All Tier 1 tools implemented and tested.

### 1.1 `cat` — File Read

- [x] `tools/cat/cat.go`:
  - Read single and multiple files
  - CDATA-wrapped content
  - Attributes: `path`, `absolute`, `size_bytes`, `lines`, `encoding`, `language`, `binary`, `mime`, `modified`
  - Flags: `-n` (line numbers), multi-file concat
  - Binary file guard: omit content, suggest `--base64`
  - Encoding detection: UTF-8, UTF-8-BOM, binary
- [x] Tests: text files, binary files, multi-file, missing files, large files, encoding edge cases

### 1.2 `grep` — Content Search

- [x] `tools/grep/grep.go`:
  - `regexp` package for pattern matching
  - Recursive directory walk with `io/fs`
  - Attributes: `pattern`, `flags`, `recursive`, `case_sensitive`, `match_type`, `searched_files`, `matched_files`, `total_matches`, `search_root`
  - Per-match: `line`, `col`, `offset_bytes`, `<before>`, `<text>`, `<after>` context
  - Flags: `-r`, `-n`, `-l`, `-i`, `-w`, `-A`, `-B`, `-C`, `-c`, `-v`, `-E`, `-F`
  - `--include`, `--exclude-dir` filters
  - Empty result: `<grep matched_files="0" total_matches="0"/>`
- [x] Tests: regex patterns, literal strings, context lines, case-insensitive, no matches, binary file skip, include/exclude filters

### 1.3 `find` — Filesystem Search

- [x] `tools/find/find.go`:
  - `filepath.WalkDir` with predicates
  - Conditions: `-name` (glob), `-type`, `-mtime`, `-size`, `-maxdepth`, `-not`, `-o` (OR)
  - Attributes per result: `path`, `absolute`, `type`, `size_bytes`, `modified`, `language`, `mime`, `depth`
  - `<condition>` elements echo search criteria
- [x] Tests: name globbing, type filtering, depth limiting, mtime, size, exclude patterns, multiple roots, OR conditions

### 1.4 `stat` — File Metadata

- [x] `tools/stat/stat.go`:
  - `os.Lstat` + `syscall.Stat_t`
  - All fields: `inode`, `links`, `device`, `permissions`, `mode_octal`, `uid`, `gid`, `owner`, `group`, `atime`, `mtime`, `ctime`, `birth`
  - Companion `_ago_s` attributes
  - Symlink handling: `-L` flag to follow
  - MIME + language enrichment
- [x] Tests: regular files, directories, symlinks, missing files, permission edge cases

### 1.5 `wc` — Count

- [x] `tools/wc/wc.go`:
  - Line, word, char, byte counting
  - `bufio.Scanner` for efficient reading
  - Per-file and total counts
  - Flags: `-l`, `-w`, `-c`, `-m`
  - Language attribute per file
- [x] Tests: empty files, single-line, multi-line, binary files, glob patterns, multi-file totals

### 1.6 `diff` — Change View

- [x] `tools/diff/diff.go`:
  - Implement Myers O(ND) diff algorithm from scratch (~150 lines)
  - Unified diff format output
  - XML elements: `<hunk>`, `<added>`, `<removed>`, `<context>`
  - Attributes: `old_start`, `old_count`, `new_start`, `new_count`, line numbers on each element
  - Summary: `added_lines`, `removed_lines`, `changed_hunks`, `identical`
  - Flags: `-u`, `--label`, `-r` (recursive), `--ignore-all-space`, `-q`
- [x] Tests: identical files, single change, multiple hunks, whitespace-only changes, recursive directory diff, empty files

### Phase 1 Acceptance Criteria

```bash
# All Tier 1 tools produce valid XML
for tool in ls cat grep find stat wc diff; do
  AICT_XML=1 ./aict $tool [args] | xmllint --noout -
done

# Each tool handles errors gracefully (no panics, structured <error> elements)
# Each tool handles empty results (valid XML, zero counts)
# Plain text passthrough works for all tools
# JSON mode works for all tools
```

---

## Phase 2 — Contextual Enrichment (Week 4-5)

**Goal**: All Tier 2 tools implemented.

### 2.1 `file` — Type Detection

- [x] `tools/file/file.go`:
  - Magic bytes + extension lookup
  - Attributes: `type`, `mime`, `category`, `language`, `charset`, `executable`
  - Flags: `-b` (brief), `-i` (MIME output)
- [x] Tests: ELF binaries, text files, images, archives, scripts with shebangs

### 2.2 `head` / `tail` — Partial Read

- [x] `tools/head/head.go`, `tools/tail/tail.go`:
  - First/last N lines or bytes
  - Attributes: `lines_requested`, `lines_returned`, `file_total_lines`, `bytes_returned`, `file_total_bytes`, `truncated`
  - Language + MIME enrichment
  - Flags: `-n`, `-c`
- [x] Tests: file smaller than request, exact match, byte mode, empty files

### 2.3 `du` / `df` — Disk Usage

- [x] `tools/du/du.go`:
  - `filepath.WalkDir` size accumulation
  - Attributes: `size_bytes`, `size_human`, `depth`
  - Flags: `-s`, `-h`, `-a`, `--max-depth`
- [x] `tools/df/df.go`:
  - `syscall.Statfs` for filesystem stats
  - Attributes: `device`, `mount`, `type`, `size_bytes`, `used_bytes`, `avail_bytes`, `use_pct`, inodes
  - Flags: `-h`
- [x] Tests: directory trees, single files, multiple filesystems

### 2.4 `realpath` / `basename` / `dirname`

- [x] `tools/realpath/realpath.go`, `tools/basename/basename.go`, `tools/dirname/dirname.go`:
  - `filepath.Abs`, `filepath.EvalSymlinks`, `filepath.Base`, `filepath.Dir`, `filepath.Ext`
  - `basename`: add `stem` and `extension` attributes
  - `realpath`: add `exists`, `type` attributes
- [x] Tests: relative paths, symlinks, non-existent paths, nested paths

### 2.5 `pwd`

- [x] `tools/pwd/pwd.go`:
  - `os.Getwd` + home directory relative path
  - Attributes: `path`, `home`, `relative_to_home`
- [x] Tests: various working directories

### 2.6 `sort` / `uniq`

- [x] `tools/sort/sort.go`:
  - Line sorting with key/field support
  - Attributes: `lines_in`, `lines_out`, `key`, `order`
  - Flags: `-n`, `-r`, `-k`, `-t`
- [x] `tools/uniq/uniq.go`:
  - Deduplication with counting
  - Attributes: `lines_in`, `lines_out`, `duplicates_removed`, `counted`
  - Flags: `-c`, `-d`, `-u`
- [x] Tests: numeric sort, reverse sort, duplicate detection, counted output

### 2.7 `cut` / `tr`

- [x] `tools/cut/cut.go`:
  - Field extraction by delimiter
  - Attributes: `delimiter`, `fields`, `lines_processed`
  - Flags: `-d`, `-f`
- [x] `tools/tr/tr.go`:
  - Character transliteration/deletion
  - Flags: `-d`, `-s`
- [x] Tests: multi-field extraction, character mapping, deletion sets

### 2.8 `env` — Environment Snapshot

- [x] `tools/env/env.go`:
  - `os.Environ()` with parsing
  - PATH parsed as `<path_entry>` list with existence checks
  - Secret detection heuristic (KEY, SECRET, TOKEN, PASSWORD, DSN, URL with auth)
  - Redaction: `present="true" redacted="true"` for secrets
  - Type classification: `path`, `path_list`, `secret`, `numeric`, `boolean`, `string`
- [x] Tests: secret redaction, PATH parsing, empty env, special characters in values

### 2.9 `id` / `whoami` / `uname` → Combined as `system`

- [x] `tools/system/system.go`:
  - `os/user.Current()`, group lookup
  - `syscall.Uname` (Linux), `runtime.GOOS/GOARCH`
  - Combined output: `<system><user>...</user><os>...</os><runtime>...</runtime></system>`
  - Distribution detection from `/etc/os-release` (Linux)
- [x] Tests: user info, group membership, OS detection, runtime info

### 2.10 `ps` — Process List

- [x] `tools/ps/ps.go`:
  - Linux: read `/proc/[pid]/stat`, `/proc/[pid]/status`, `/proc/[pid]/cmdline`
  - Darwin: `syscall.SysctlRaw` fallback
  - Attributes: `pid`, `ppid`, `user`, `uid`, `cpu_pct`, `mem_pct`, `vsz_kb`, `rss_kb`, `state`, `state_desc`, `started`, `command`, `args`, `exe`
  - Flags: `aux`, `-ef`, `-p`, `--sort`
- [x] Tests: process listing, specific PID, state decoding, command parsing

### 2.11 `md5sum` / `sha256sum` → Combined as `checksums`

- [x] `tools/checksums/checksums.go`:
  - `crypto/md5`, `crypto/sha256`, `crypto/sha1`
  - `io.MultiWriter` for parallel hashing
  - Attributes: `md5`, `sha256`, `sha1`, `size_bytes`
  - Flags: `-c` (verify against checksum file)
- [x] Tests: single file, multi-file, verification mode, empty files, binary files

### Phase 2 Acceptance Criteria

```bash
# All Tier 2 tools produce valid XML
# Secret redaction works in env output
# Process listing works on current platform
# Checksums match known values
# Disk usage numbers are reasonable
```

### MCP Server (Bonus)

- [x] `cmd/mcp/server.go`:
  - MCP protocol implementation
  - Exposes all tools as callable MCP functions
  - JSON output for tool results
- [x] Tests: MCP protocol handling, tool invocation

---

## Phase 3 — Polish & Performance (Week 6)

**Goal**: Production-ready quality, performance benchmarks, documentation.

### 3.1 Performance

- [ ] Benchmark all tools against GNU coreutils:
  - `ls` on 1000-file directory
  - `grep` on 100k-line file
  - `find` on deep directory tree
  - `cat` on 10MB file
  - `diff` on large files
- [ ] Optimize hot paths:
  - Buffered I/O for large files
  - Parallel file scanning in `grep` (goroutines + channels)
  - Streaming XML output (don't buffer entire result)
- [ ] Memory profiling: `go test -bench=. -memprofile=mem.out`
- [ ] CPU profiling: `go test -bench=. -cpuprofile=cpu.out`

### 3.2 Cross-platform

- [ ] Linux: primary target, full feature support
- [ ] macOS: `syscall.Uname` alternative, `/proc` alternative for `ps`
- [ ] Windows: subset support (ls, cat, stat, wc, find, diff work; ps, df, uname need adaptation)
- [ ] Build matrix: `GOOS=linux GOARCH=amd64`, `GOOS=darwin GOARCH=amd64`, `GOOS=darwin GOARCH=arm64`, `GOOS=windows GOARCH=amd64`

### 3.3 Documentation

- [ ] `README.md`:
  - Project overview, motivation, installation
  - Quick start with XML output examples
  - Tool inventory table
  - Comparison with GNU coreutils
- [ ] `docs/` directory:
  - Per-tool documentation with XML schema reference
  - Migration guide from GNU coreutils
  - Integration guide for AI coding agents
- [ ] `CHANGELOG.md`: Keep a Changelog format
- [ ] `CONTRIBUTING.md`: How to add new tools

### 3.4 Packaging

- [ ] `go install github.com/synseqack/aict@latest` works
- [ ] GitHub Actions CI:
  - `go test ./...` on push
  - `go vet ./...` on push
  - Build artifacts for linux/amd64, darwin/amd64, darwin/arm64, windows/amd64
  - Release on tag
- [ ] Docker image: `FROM scratch` with single binary
- [ ] Shell completion scripts (bash, zsh)

### Phase 3 Acceptance Criteria

```bash
# Performance: no tool is >10x slower than GNU equivalent for typical codebases
# Cross-platform: all Tier 1 tools work on Linux, macOS, Windows
# CI passes on all platforms
# Docker image < 10MB (static Go binary)
# Documentation covers all tools with XML examples
```

---

## Phase 4 — Future Enhancements (Post-MVP)

- [x] `--json` output mode for all tools (mirror XML structure)
- [ ] `rg` (ripgrep) integration: spawn `rg --json` if available, parse and re-emit as XML
- [ ] `git` subcommands: `git status`, `git diff`, `git log`, `git ls-files`, `git blame` with XML output
- [ ] Tree-sitter integration for `grep` function-name enrichment (optional, cgo)
- [ ] `--stream` mode: emit XML elements as they're discovered (SAX-like, for huge directories)
- [x] MCP server wrapper: expose all tools as MCP tools for direct AI agent consumption
- [ ] `aict doctor`: self-diagnostic command that checks PATH, permissions, platform support

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| Myers diff algorithm complexity | Medium | Reference implementation available; ~150 lines |
| macOS `ps` without `/proc` | Medium | Use `syscall.SysctlRaw` or `os/exec` fallback |
| Windows path semantics | Low | `filepath` package handles most cases; skip Unix-only tools |
| XML output size for huge dirs | Low | Streaming mode in Phase 4; pagination flags |
| `grep` performance vs ripgrep | Medium | Acceptable for normal codebases; ripgrep integration in Phase 4 |

---

## Timeline Summary

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| 0 — Foundation | Week 1 | Project scaffold + `ls` working |
| 1 — Core Reads | Week 2-3 | All 7 Tier 1 tools working |
| 2 — Enrichment | Week 4-5 | All 15 Tier 2 tools working |
| 3 — Polish | Week 6 | Benchmarks, cross-platform, docs, CI |
| 4 — Future | Post-MVP | JSON mode, git tools, MCP server |

**Total MVP**: 6 weeks with focused development.
