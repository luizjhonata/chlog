# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- initial project scaffold with Go module, Cobra CLI, Makefile, and CI/CD pipeline
- core types: `Config` with YAML loading and upward search, `Change` with marshal/unmarshal and sorting
- `new` command: create changelog fragments with `--kind` and `--body` flags
- `batch` command: compile fragments into version file with explicit semver, bump, or auto resolution
- `merge` command: insert version files into CHANGELOG.md with correct ordering and cleanup
- `check` command: verify unreleased fragments exist for CI gating
- README.md with install, usage, configuration, and CI integration docs
