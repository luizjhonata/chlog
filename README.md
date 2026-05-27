# chlog

Fragment-based changelog management for teams. Each change gets its own file — no more merge conflicts on `CHANGELOG.md`.

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

```bash
go install github.com/luizjhonata/chlog@latest
```

Or build from source:

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
chlog batch 1.0.0          # explicit version
chlog batch minor           # bump minor from latest
chlog batch auto            # infer bump from kind→level mapping
```

This creates `.changes/v1.0.0.md` with Keep a Changelog format and deletes consumed fragments.

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

### Version

```bash
chlog --version
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

## License

[MIT](LICENSE)
