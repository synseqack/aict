# Tools Reference

Complete reference for all 33 `aict` commands. Each tool outputs structured XML/JSON by default, with optional plain text mode for compatibility.

## Common Flags

Every tool supports these global output flags:

| Flag | Description |
|------|-------------|
| `--xml` | XML output (default if `AICT_XML=1`) |
| `--json` | JSON output |
| `--plain` | Plain text output |
| `--pretty` | Pretty-printed output |

## awk

Extract fields and apply pattern-action rules to file contents

```bash
aict awk [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--fieldsep` | string | Field separator (default: whitespace) |
| `--program` | string | Awk program (e.g. `{print $1}`) |

## basename

Print filename portion of file paths

```bash
aict basename [flags] [arguments...]
```

## cat

Read and output file contents with metadata

```bash
aict cat [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--linenumbers` | boolean | Show line numbers |

## completions

Generate shell completion scripts for aict

```bash
aict completions [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--shell` | string | Shell type: `bash`, `zsh`, or `fish` |

## checksums

Calculate MD5, SHA1, and SHA256 checksums for files

```bash
aict checksums [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--algorithms` | string | Hash algorithm (md5, sha1, sha256) |
| `--verify` | boolean | Verify checksums from file |

## cut

Cut out sections of each line from files

```bash
aict cut [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--characters` | string | Select characters (e.g., 1-10) |
| `--delimiter` | string | Field delimiter (default: tab) |
| `--fields` | string | Select fields (e.g., 1,3-5) |
| `--onlydelim` | boolean | Only print lines with delimiter |

## df

Display disk filesystem usage statistics

```bash
aict df [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--humansize` | boolean | Show sizes in human-readable format |

## diff

Compare two files or directories and show differences

```bash
aict diff [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--context` | integer | Number of context lines |
| `--ignoreallspace` | boolean | Ignore all whitespace changes |
| `--labelnew` | string | Label for new file in diff |
| `--labelold` | string | Label for old file in diff |
| `--quiet` | boolean | Output only whether files differ |
| `--recursive` | boolean | Compare directories recursively |
| `--unified` | boolean | Use unified diff format |

## dirname

Print directory portion of file paths

```bash
aict dirname [flags] [arguments...]
```

## doctor

Run diagnostics to check aict installation and environment

```bash
aict doctor [flags] [arguments...]
```

## du

Estimate disk usage of directories and files

```bash
aict du [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--all` | boolean | Count all files, not just directories |
| `--humansize` | boolean | Show sizes in human-readable format |
| `--maxdepth` | integer | Maximum depth to show entries |
| `--summarize` | boolean | Show only total for each argument |

## env

Display environment variables with types and redaction

```bash
aict env [flags] [arguments...]
```

## file

Determine file type using MIME detection and content analysis

```bash
aict file [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--brief` | boolean | Show brief file type only |
| `--mime` | boolean | Show MIME type only |

## find

Find files by name, type, or modification time

```bash
aict find [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--invert` | boolean | Invert match conditions |
| `--maxdepth` | integer | Maximum directory depth |
| `--mtime` | integer | Modified within N days |
| `--name` | string | File name pattern (supports * and ?) |
| `--or` | boolean | OR between conditions |
| `--size` | integer | File size in bytes |
| `--type` | string | File type: f (regular), d (directory), l (symlink) |

## git

Run git subcommands (status, diff, log, ls-files, blame, show)

```bash
aict git [flags] [arguments...]
```

## grep

Search for patterns in files with line numbers and context

```bash
aict grep [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--aftercontext` | integer | Number of context lines after match |
| `--beforecontext` | integer | Number of context lines before match |
| `--caseinsensitive` | boolean | Case insensitive search |
| `--contextlines` | integer | Number of context lines around match |
| `--countonly` | boolean | Count matches only, don't show content |
| `--excludedir` | string | Exclude directories matching pattern |
| `--extendedregex` | boolean | Use extended regular expressions |
| `--fileswithmatches` | boolean | Show only file names with matches |
| `--fixedstrings` | boolean | Treat pattern as literal string |
| `--include` | string | Include files matching pattern (e.g., *.go) |
| `--invertmatch` | boolean | Invert match - show non-matching lines |
| `--linenumbers` | boolean | Show line numbers |
| `--maxcount` | integer | Stop after N matches |
| `--pattern` | string | Search pattern (regex or literal) |
| `--recursive` | boolean | Search recursively in directories |
| `--wordmatch` | boolean | Match whole words only |

## head

Display the first N lines of a file

```bash
aict head [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--bytes` | integer | Number of bytes to show |
| `--lines` | integer | Number of lines to show |

## jq

Extract values from JSON files using path expressions

```bash
aict jq [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--compact` | boolean | Compact output (no pretty-print) |
| `--path` | string | Path expression (e.g. `.key`, `.[0]`, `.[].field`) |
| `--raw` | boolean | Output raw strings without JSON quotes |

## ls

List directory contents with file metadata including permissions, size, and modification time

```bash
aict ls [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--all` | boolean | Show hidden files (starting with .) |
| `--almostall` | boolean | Show almost all (exclude . and ..) |
| `--recursive` | boolean | List subdirectories recursively |
| `--reverse` | boolean | Reverse sort order |
| `--sorttime` | boolean | Sort by modification time, newest first |

## md5sum

Calculate MD5 checksum for files

```bash
aict md5sum [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--algorithms` | string | Hash algorithm (md5, sha1, sha256) |
| `--verify` | boolean | Verify checksums from file |

## ps

List running processes with details

```bash
aict ps [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--all` | boolean | Show all processes |
| `--full` | boolean | Show full command details |
| `--pid` | string | Filter by PID |
| `--sortby` | string | Sort by field (e.g., pid, cpu, mem) |

## pwd

Print current working directory

```bash
aict pwd [flags] [arguments...]
```

## sed

Apply sed-style transformations to file contents

```bash
aict sed [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--extendedregex` | boolean | Use extended regular expressions (`-E`) |
| `--script` | string | Sed script to apply (e.g. `s/foo/bar/g`) |
| `--suppress` | boolean | Suppress default output (`-n`) |

Supported commands: `s` (substitute), `d` (delete), `p` (print), `q` (quit). Address forms: line number, `/regex/`, `N,M` range, `$` (last line).

## realpath

Print resolved absolute paths

```bash
aict realpath [flags] [arguments...]
```

## sha1sum

Calculate SHA1 checksum for files

```bash
aict sha1sum [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--algorithms` | string | Hash algorithm (md5, sha1, sha256) |
| `--verify` | boolean | Verify checksums from file |

## sha256sum

Calculate SHA256 checksum for files

```bash
aict sha256sum [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--algorithms` | string | Hash algorithm (md5, sha1, sha256) |
| `--verify` | boolean | Verify checksums from file |

## tar

List or extract contents from tar archives (.tar, .tar.gz, .tgz, .tar.bz2)

```bash
aict tar [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--extract` | string | Extract a specific file path to stdout |
| `--list` | boolean | List archive contents |

## sort

Sort lines of text files

```bash
aict sort [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--delimiter` | string | Field delimiter (default: tab) |
| `--key` | integer | Sort by field number (1-based) |
| `--numeric` | boolean | Sort numerically |
| `--outputfile` | string | Write output to file |
| `--reverse` | boolean | Sort in reverse order |
| `--unique` | boolean | Remove duplicate lines |

## stat

Display detailed file metadata including timestamps, permissions, and ownership

```bash
aict stat [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--followsymlinks` | boolean | Follow symlinks and show target file info |

## system

Display system information including user, OS, and runtime details

```bash
aict system [flags] [arguments...]
```

## tail

Display the last N lines of a file

```bash
aict tail [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--bytes` | integer | Number of bytes to show |
| `--follow` | boolean | Follow file updates in real-time |
| `--lines` | integer | Number of lines to show |

## tr

Translate, squeeze, or delete characters from stdin

```bash
aict tr [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--delete` | boolean | Delete characters in set1 |
| `--squeeze` | boolean | Squeeze repeated characters |
| `--translate` | boolean | Translate characters |

## uniq

Report or filter out repeated lines

```bash
aict uniq [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--count` | boolean | Prefix lines by number of occurrences |
| `--duplicates` | boolean | Only show duplicate lines |
| `--ignorecase` | boolean | Case insensitive comparison |
| `--unique` | boolean | Only show unique lines |

## wc

Count lines, words, and bytes in files

```bash
aict wc [flags] [arguments...]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--allfiles` | boolean | Count all files including hidden |
| `--bytes` | boolean | Count bytes |
| `--lines` | boolean | Count lines |
| `--maxlines` | boolean | Show maximum line length |
| `--words` | boolean | Count words |

