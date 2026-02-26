# Contributing to mbview-go

Thanks for contributing.

## Development setup

1. Install Go (see `go.mod` for minimum version).
2. Clone the repo.
3. Run tests:

```bash
go test ./...
```

## Pull request checklist

- Keep changes focused and atomic.
- Include tests for behavior changes.
- Update `README.md` if CLI flags or behavior change.
- Ensure CI passes.

## Commit style

Use clear commit messages, for example:

- `feat: add ...`
- `fix: handle ...`
- `test: cover ...`
- `docs: update ...`

## Reporting issues

When opening a bug report, include:

- OS and architecture
- Go version (if building from source)
- command used
- sample MBTiles metadata (if possible)
- expected and actual behavior
