# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

GitHub release notes are generated automatically by GoReleaser from
Conventional Commit messages; this file tracks human-curated highlights.

## [Unreleased]

### Added
- Initial release of ginkgoleaf, a Ginkgo v2 output formatter with eight
  output formats: `tree`, `jest`, `markdown`, `github`, `gitlab`, `text`,
  `shell`, and `tap`.
- Structured formats (`tree`, `github`, `gitlab`, `markdown`) render the
  spec hierarchy by structure, not execution order: a container that
  re-enters non-contiguously under Ginkgo's `--randomize-all` is merged
  into a single node rather than appearing many times.
- Suite-level setup/teardown nodes (`BeforeSuite`, `AfterSuite`, their
  `Synchronized` variants, and suite-scoped `DeferCleanup`) are omitted when
  they pass — so spec counts match Ginkgo's own totals rather than the raw
  report's node count — and surfaced, labelled by node type (e.g.
  `[BeforeSuite]`), when they fail.
- Release artifacts ship SBOMs (SPDX JSON via syft), keyless cosign
  signatures, and GitHub-native SLSA build provenance attestations.
- CI `govulncheck` vulnerability gate.
- `SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`.
