# Pull Request

## Description

<!-- Clear, concise description of what this PR does -->

## Type of Change

- [ ] Bug fix
- [ ] New tool
- [ ] Tool enhancement
- [ ] Refactor
- [ ] Documentation
- [ ] Performance
- [ ] CI/CD
- [ ] Other: ______

## Related Issue

<!-- Link to issue(s) this PR addresses, e.g., Fixes #123 -->

Fixes #

## Checklist

- [ ] `go build -o aict ./cmd/aict` succeeds
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] `gofmt -w .` applied
- [ ] XML output is valid for the new/changed tool
- [ ] JSON output is valid (`--json` flag)
- [ ] Plain text output works (`--plain` flag)
- [ ] Error handling returns structured `<error>` elements (no stderr)
- [ ] All paths in output are absolute
- [ ] All sizes are in bytes with `_human` companion
- [ ] All timestamps are Unix epoch integers with `_ago_s` companion
- [ ] Empty results produce valid XML
- [ ] No external dependencies added (stdlib only)
- [ ] Tests added/updated for new functionality

## Tool Output Sample

<!-- If adding/modifying a tool, paste sample XML output -->

```xml
<!-- sample output -->
```

## Notes

<!-- Any additional context, trade-offs, or follow-up items -->
