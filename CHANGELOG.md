# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-07-28

### Added

- chlog new --breaking flag to mark backward-incompatible changes and force a major bump

### Changed

- auto bump no longer maps Changed/Removed to major; major now requires an explicit breaking fragment, per SemVer

## [0.3.1] - 2026-07-10

### Fixed

- emit fragment `time` field single-quoted to satisfy YAML-quoting linters (output unchanged)

### Security

- pin GitHub Actions to full commit SHAs and add Dependabot cooldown to harden CI against supply-chain attacks

## [0.3.0] - 2026-06-11

### Changed

- `batch auto` no longer graduates a 0.x project to 1.0.0; a breaking change bumps the minor until 1.0.0 is reached explicitly

### Fixed

- resolve `batch` relative version bumps from git tags and CHANGELOG.md, not only leftover version files (bumps no longer reset to 0.0.0 after a release)

## [0.2.0] - 2026-06-11

### Added

- `chlog ai setup` to inject changelog rules into detected AI assistant instruction files
- add `chlog hook install` and `chlog hook uninstall` commands for pre-commit hook management

### Changed

- replace per-repo hook with global `core.hooksPath` hook (default) and local injection mode (`--local`) for projects with existing hook managers
- extract `FindConfig(startDir)` pure function from `FindConfigUpward()` for testability
- quote kind and body values with single quotes in generated fragment files
- improve README with badges, features, positioning, install methods, and contributing section

## [0.1.0] - 2026-05-27

### Added

- initial project scaffold with Go module, Cobra CLI, Makefile, and CI/CD pipeline
- core types: `Config` with YAML loading and upward search, `Change` with marshal/unmarshal and sorting
- `new` command: create changelog fragments with `--kind` and `--body` flags
- `batch` command: compile fragments into version file with explicit semver, bump, or auto resolution
- `merge` command: insert version files into CHANGELOG.md with correct ordering and cleanup
- `check` command: verify unreleased fragments exist for CI gating
- README.md with install, usage, configuration, and CI integration docs
- make release target to automate batch and merge workflow
- SonarCloud integration with coverage reporting
- security policy with responsible disclosure instructions
- issue templates for bug reports and feature requests
- Dependabot config for Go modules and GitHub Actions weekly updates
- bootstrap `.changes/` directory with `.chlog.yaml` config — project now uses its own tool

### Changed

- replace shared pipeline delegation with local workflow using `chlog check` instead of CHANGELOG.md check
- update CLAUDE.md with correct testing, lint, commit, and changelog conventions

### Fixed

- add `v` prefix to release tags for Go toolchain compatibility
- skip fragment check on bump branches in CI pipeline
