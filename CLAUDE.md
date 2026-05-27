# CLAUDE.md

## What This Project Does

chlog is a fragment-based changelog CLI. Each PR creates a YAML fragment in `.changes/unreleased/`; at release time, fragments are compiled into CHANGELOG.md. Designed to eliminate merge conflicts on CHANGELOG.md in multi-developer environments.

It is a local filesystem tool — no Git, no APIs, no provider integration. Works in any repo.

## Build & Development Commands

```bash
make build    # compile binary to bin/chlog
make debug    # compile with debug symbols
make run      # go run .
make install  # build + copy to ~/.local/bin
make lint     # run linters (from shared pipelines)
make test     # run tests (from shared pipelines)
make sast     # run security analysis (from shared pipelines)
```

Single test:
```bash
go test -tags unit -run "TestName" ./cmd/
go test -tags unit -run "TestName" ./internal/
```

## Architecture

Flat, idiomatic Go. No CQRS, no DI container.

- `main.go` — entry point, version injection via ldflags
- `cmd/` — Cobra commands with business logic inline
- `internal/` — shared types (Config, Change) with Go-native import protection

## Key External Libraries

- `github.com/spf13/cobra` — CLI framework
- `github.com/Masterminds/semver/v3` — semantic versioning
- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/stretchr/testify` — test assertions (test only)

## Testing Conventions

- Build tag: `//go:build unit` on all test files
- BDD structure: `// given`, `// when`, `// then`
- Test names: `t.Run("should ... when ...", ...)`
- `t.Parallel()` on every test
- `t.TempDir()` for filesystem isolation
- testify/assert + testify/require

## Configuration

`.chlog.yaml` searched from cwd upward. Defaults provided when not found.

## Commands

| Command | Purpose |
|---------|---------|
| `chlog new --kind <kind> --body "<text>"` | Create a changelog fragment |
| `chlog batch <version\|major\|minor\|patch\|auto>` | Compile fragments into versioned file |
| `chlog merge` | Append version files to CHANGELOG.md |
| `chlog check` | Validate fragments exist (CI) |
| `chlog --version` | Print version |
