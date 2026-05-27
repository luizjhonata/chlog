# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
