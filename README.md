# chlog

[![CI](https://github.com/luizjhonata/chlog/actions/workflows/default.yaml/badge.svg)](https://github.com/luizjhonata/chlog/actions/workflows/default.yaml)
[![Release](https://img.shields.io/github/v/release/luizjhonata/chlog)](https://github.com/luizjhonata/chlog/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/luizjhonata/chlog.svg)](https://pkg.go.dev/github.com/luizjhonata/chlog)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Fragment-based changelog management for teams. Each change gets its own file — no more merge conflicts on `CHANGELOG.md`.

## Features

- **No merge conflicts** — each PR creates a uniquely-named YAML file instead of editing a shared `CHANGELOG.md`
- **Keep a Changelog format** — outputs standard `[Unreleased]`, `### Added`, `### Fixed` sections
- **Automatic version bumps** — infer the next semver version from change kinds (Added → minor, Fixed → patch)
- **CI gating** — `chlog check` enforces that every PR includes a changelog entry
- **Git hook** — `chlog hook install` sets up a global pre-commit hook that validates fragments in any chlog-enabled repo
- **AI assistant integration** — `chlog ai setup` teaches your installed AI assistants to create fragments automatically
- **Minimal and fast** — single binary, no runtime dependencies

## Why chlog?

- **Six commands** — `new`, `batch`, `merge`, `check`, `hook`, `ai`. Nothing else to learn
- **Explicit over implicit** — developers write what changed, not structured commit messages
- **~500 lines of Go** — easy to audit, fork, and contribute
- **Zero configuration required** — sensible defaults, optional `.chlog.yaml` for customization

## How It Works

Instead of editing `CHANGELOG.md` directly, each PR adds a small YAML fragment to `.changes/unreleased/`. At release time, fragments are compiled into the changelog. This eliminates merge conflicts in multi-developer workflows.

```
.changes/
├── unreleased/
│   ├── 1748359200-a1b2.yaml    # fragment per change
│   └── 1748359300-c3d4.yaml
├── v1.0.0.md                   # compiled version file (intermediate)
└── v1.1.0.md
```

## Install

### Pre-built binaries

Download from [GitHub Releases](https://github.com/luizjhonata/chlog/releases/latest). Binaries are available for Linux, macOS, and Windows (amd64/arm64).

### Go install

```bash
go install github.com/luizjhonata/chlog@latest
```

### Build from source

```bash
git clone git@github.com:luizjhonata/chlog.git
cd chlog
make install    # builds and copies to ~/.local/bin
```

## Usage

### Create a fragment

```bash
chlog new --kind Added --body "user authentication via OAuth2"
chlog new --kind Fixed --body "null pointer on empty config"
```

Kind matching is case-insensitive. The fragment is written to `.changes/unreleased/<timestamp>-<random>.yaml`.

### Compile fragments into a version file

```bash
chlog batch 1.0.0   # explicit version
chlog batch minor   # bump minor from latest
chlog batch auto    # infer bump from kind→level mapping
```

This creates `.changes/v1.0.0.md` with Keep a Changelog format and deletes consumed fragments.

The base version is resolved from the highest of: existing version files, git tags, and `CHANGELOG.md` headings.

While the project is below `1.0.0`, `auto` never graduates to `1.0.0` on its own — a breaking change bumps the minor instead (e.g. `0.2.0` → `0.3.0`), per SemVer's unstable-`0.x` rule. Reaching `1.0.0` is a deliberate, explicit step (`chlog batch major` or `chlog batch 1.0.0`).

### Merge into CHANGELOG.md

```bash
chlog merge
```

Inserts all version files into `CHANGELOG.md` in descending order, preserving existing content. Deletes consumed version files.

### Verify fragments exist (CI)

```bash
chlog check
```

Exits 0 if unreleased fragments exist, exits 1 otherwise. Use in CI to enforce that every PR includes a changelog entry.

### Git hook

```bash
chlog hook install              # global hook (install once, works in every repo)
chlog hook install --local      # inject into current repo's existing hook (e.g., Husky)
chlog hook uninstall            # remove global hook
chlog hook uninstall --local    # remove injected block from current repo
```

**Global mode (default)** — sets `core.hooksPath` so the hook runs in every repo. Detects `.chlog.yaml` before acting — repos without chlog are unaffected. Chains per-repo hooks automatically so existing hooks keep working.

**Local mode (`--local`)** — appends a chlog block to the current repo's pre-commit hook. Use this when the project already manages hooks via Husky, Lefthook, or similar. The block is injected without modifying existing hook content.

- Idempotent — re-installing when the hook is already present is a no-op
- Safe — refuses to override an existing `core.hooksPath` unless `--force` is passed
- `--force` re-installs hooks (useful after a chlog update)

### AI assistant integration

```bash
chlog ai setup           # inject chlog rules into detected assistants
chlog ai setup --force   # re-inject the block if it already exists
```

Detects which AI coding assistants are installed on your machine and injects a
mandatory changelog rule into the matching project instruction file, so the
assistant creates fragments automatically whenever you work in a chlog repo.

| Assistant | Detected via | Instruction file |
|-----------|--------------|------------------|
| Claude Code | `claude` in `PATH` or `~/.claude/` | `CLAUDE.md` |
| OpenAI Codex | `codex` in `PATH` or `~/.codex/` | `AGENTS.md` |
| Cursor | `~/.cursor/` or app config dir | `AGENTS.md` |
| Gemini CLI | `gemini` in `PATH` or `~/.gemini/` | `GEMINI.md` |
| GitHub Copilot | `~/.config/github-copilot/` | `.github/copilot-instructions.md` |
| Windsurf | `~/.windsurf/`, `~/.codeium/windsurf/` or app config dir | `.windsurf/rules/chlog.md` |

- The injected rule is conditional — it only applies in repos that have a `.chlog.yaml`, so it is safe to commit and share
- Idempotent — the block is delimited by `<!-- chlog:start -->` / `<!-- chlog:end -->` and existing content is preserved
- If no supported assistant is detected, the command does nothing
- `--force` re-injects the block (useful after a chlog update)

#### Adding the rule manually

If detection misses your assistant, or you prefer to set it up by hand, copy the
block below into your assistant's instruction file (`CLAUDE.md`, `AGENTS.md`,
`GEMINI.md`, `.github/copilot-instructions.md`, `.windsurf/rules/chlog.md`, …).
The `Valid kinds` line should mirror the `kinds` in your `.chlog.yaml` (the
values below are the defaults).

```markdown
<!-- chlog:start -->
## Changelog (chlog) — MANDATORY

If the repository you are working in uses chlog (a `.chlog.yaml` or `.chlog.yml`
config file, or a `.changes/` directory, exists at the project root), the
following is binding and ALWAYS applies: whenever you make ANY change, you MUST
create a changelog fragment as part of the same change — automatically, without
being asked, before committing.

- Do NOT edit CHANGELOG.md directly; it is generated from fragments.
- Create the fragment with:
  `chlog new --kind <Kind> --body "<imperative description>"`
- Valid kinds: Added, Changed, Deprecated, Removed, Fixed, Security
- Choose the kind that best matches the change (e.g., new feature → Added,
  bug fix → Fixed, behavior change → Changed, removal → Removed, security fix → Security).
- Fragments are YAML files in `.changes/unreleased/`; stage them with your commit.
- `chlog check` fails the build when a fragment is missing — never skip it.
<!-- chlog:end -->
```

## Release Flow

```
1. Developer creates fragments during PR work
   $ chlog new --kind Added --body "new feature"

2. At release time, compile fragments
   $ chlog batch auto

3. Review the generated version file, then merge
   $ chlog merge

4. Commit CHANGELOG.md and tag the release
```

## Configuration

Create `.chlog.yaml` in your project root. If not found, defaults are used.

```yaml
changesDir: .changes
unreleasedDir: unreleased
changelogPath: CHANGELOG.md
versionFormat: '## [{{.Version}}] - {{.Time.Format "2006-01-02"}}'
kindFormat: '### {{.Kind}}'
changeFormat: '- {{.Body}}'
kinds:
  - label: Added
    auto: minor
  - label: Changed
    auto: major
  - label: Deprecated
    auto: minor
  - label: Removed
    auto: major
  - label: Fixed
    auto: patch
  - label: Security
    auto: patch
```

The `auto` field maps each kind to a semver bump level, used by `chlog batch auto`.

Format fields use Go templates. Available variables:

| Template | Variables |
|----------|-----------|
| `versionFormat` | `.Version`, `.Time` |
| `kindFormat` | `.Kind` |
| `changeFormat` | `.Body` |

## CI Integration

Add `chlog check` to your PR pipeline to require a changelog fragment:

```yaml
# GitHub Actions example
- name: Verify changelog fragment
  run: chlog check
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, commit conventions, and code style guidelines.

Found a bug or have a feature request? [Open an issue](https://github.com/luizjhonata/chlog/issues).

## License

[MIT](LICENSE)
