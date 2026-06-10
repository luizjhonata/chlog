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
```

`make lint`, `make test`, and `make sast` require the [shared pipelines](https://github.com/rios0rios0/pipelines) repo cloned at `~/Development/github.com/rios0rios0/pipelines`. Without it, use the commands below directly.

Lint (CI uses golangci-lint v2 with external config — local v1 misses issues):
```bash
~/go/bin/golangci-lint run --config ~/Development/github.com/rios0rios0/pipelines/global/scripts/languages/golang/golangci-lint/.golangci.yml ./...
```

Tests:
```bash
go test -tags unit -count=1 -v ./...
go test -tags unit -run "TestName" ./cmd/        # single test
go test -tags unit -run "TestName" ./internal/   # single test
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
- Do NOT use `t.Parallel()` in `cmd/` tests — they use `os.Chdir` which is process-global
- `t.Parallel()` is used in `internal/` tests
- `t.TempDir()` for filesystem isolation
- testify/assert + testify/require

## Commit Conventions

- Format: `type(scope): concise imperative description`
- Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`
- Trailer: `Signed-off-by: Name <email>` (no Co-Authored-By)
- Max 72 chars title, lowercase first letter after colon, no period at end

## Changelog Workflow

This project uses its own tool for changelog management. Do NOT edit `CHANGELOG.md` directly.

```bash
# 1. For every PR, create a fragment:
chlog new --kind Added --body "description of the change"

# 2. At release time, compile fragments into a version file:
chlog batch auto

# 3. Merge version file into CHANGELOG.md:
chlog merge
```

Fragment files live in `.changes/unreleased/` as YAML:
```yaml
kind: Added
body: description of the change
time: 2026-05-27T12:00:00Z
```

Every PR MUST include at least one fragment. Use `chlog check` in CI to enforce this.

## Configuration

`.chlog.yaml` in project root. Searched upward from cwd. Defaults used when not found.

## Commands

| Command | Purpose |
|---------|---------|
| `chlog new --kind <kind> --body "<text>"` | Create a changelog fragment |
| `chlog batch <version\|major\|minor\|patch\|auto>` | Compile fragments into versioned file |
| `chlog merge` | Append version files to CHANGELOG.md |
| `chlog check` | Validate fragments exist (CI) |
| `chlog hook install [--local] [--force]` | Install a global or local pre-commit hook that runs `chlog check` |
| `chlog hook uninstall [--local]` | Remove the chlog-managed hook (global or local) |
| `chlog ai setup [--force]` | Inject chlog rules into detected AI assistant instruction files |
| `chlog --version` | Print version |
