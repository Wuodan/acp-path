# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-10-30

### Added

- ACP stdio proxy that translates filesystem paths between Windows and WSL.
- Support for ACP v1 and v2 schema-aware path mapping, with optional mapping of
  absolute paths in unknown fields.
- `acp-wsl-debug` command for logging complete ACP traffic and path-mapping
  decisions.
- GitHub Action for resolving a release tag and optionally downloading release
  assets.
- Automated release artifacts for Windows and validated WinGet manifests.
