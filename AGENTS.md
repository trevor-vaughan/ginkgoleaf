# AGENTS.md — ginkgoleaf

Guidance for AI agents working in this repository. For the human-facing
build/test/contribution workflow, see [`CONTRIBUTING.md`](CONTRIBUTING.md);
this file documents the conventions that keep linting and CI green and that
are easy to get wrong.

`ginkgoleaf` is a third-party Ginkgo v2 output formatter (Go): an in-suite
library and a standalone CLI that renders Ginkgo's JSON report in several
human- and machine-readable formats.

## Repository layout: tracked code vs. vendored tooling

Only these top-level directories hold **our** tracked source:
`cmd/`, `render/`, `internal/`, `pkg/`, `gomega/`, `tools/`, `fixtures/`,
`tests/`, plus `.github/` and the root Go/Taskfile/doc files.

The following are **vendored or generated** and are git-ignored:

| Path                 | What it is                                              |
|----------------------|---------------------------------------------------------|
| `.lola/`             | Agent skill/pack modules installed by the `lola` CLI    |
| `.opencode/`         | Host agent tooling                                      |
| `megalinter-reports/`| MegaLinter output (findings, SARIF, logs)               |
| `bin/`, `dist/`      | Build artifacts                                         |

**Never edit files under the vendored/generated paths.** They are installed
by external CLIs; edits are untracked and are clobbered on the next reinstall.
If a linter reports findings inside them, the fix is to keep them excluded
(see below), not to modify them.

## Linting

MegaLinter is the umbrella linter. Its **file-level** linters (one per
language) operate on the git-tracked file list, so `.gitignore` keeps the
vendored directories above out of their scope — git-ignore a new
vendored/generated directory and those linters skip it.

**Repository-scoped scanners do not honor `.gitignore`.** The `REPOSITORY_*`
linters — Trivy, gitleaks, secretlint, and friends — walk the filesystem tree
directly, so they descend into git-ignored, untracked directories such as
`.lola/` and report findings there regardless of `.gitignore`. Proof: a
`REPOSITORY_TRIVY` run flagged
`.lola/modules/review-council/.lola-eval/tests/case-005-py-meta/starter/Dockerfile`
(DS-0002 no non-root `USER`, DS-0026 no `HEALTHCHECK`) even though `.lola/`
is git-ignored and the file is untracked.

Those particular findings are **intentional**: the `.lola-eval/.../starter/`
trees are review-council evaluation fixtures with one flaw planted per
reviewer persona (the Dockerfile's missing `USER`/`HEALTHCHECK` is the SRE
persona's target — see the case `rubric.md`). They are false positives for
*our* repo and must never be "fixed" by editing the fixture.

To keep a vendored tree out of a repository-scoped scanner, exclude it at the
**MegaLinter config layer**, not via `.gitignore` and not by editing the
vendored file. For Trivy this is `--skip-dirs` (verified to clear the findings
above):

    # .mega-linter.yml
    REPOSITORY_TRIVY_ARGUMENTS: ["--skip-dirs", ".lola/**", "--skip-dirs", ".opencode/**"]

Do **not** suppress these with a repo-root `.trivyignore`: bare-ID entries
blanket-disable the rule repo-wide and would silently hide a genuine flaw in
a real shipped Dockerfile later. Trivy's path-scoped `.trivyignore.yaml` is
not auto-detected (it needs an explicit `--ignorefile`), so it is dead config
without the matching MegaLinter argument. Because the MegaLinter config syncs
org-wide (`sync-config.yml`), treat any such change as shared state: propose
it for human review rather than committing it unilaterally.

Per-language conventions that surface as findings if missed:

- **Shell (shellcheck):** use `[[ ... ]]`, not `[ ... ]` (SC2292). Add a
  default `*)` arm to `case` statements. Quote and use `find -print0 | xargs
  -0` for filename safety.
- **YAML (yamllint):** a scalar **value** that must stay the string `off`,
  `on`, `yes`, or `no` has to be quoted, or yamllint's `truthy` rule rejects
  it — e.g. `GOWORK: "off"` in `release.yml`. (Workflow trigger *keys* like
  `on:` are allowed by the active config; only ambiguous values need quoting.)
- **GitHub Actions (zizmor):**
  - Set `persist-credentials: false` on every `actions/checkout` step — the
    job token is otherwise written into `.git/config` and can leak through
    later steps or uploaded artifacts (`artipacked`). CI here only reads code,
    and GoReleaser authenticates via an explicit `GITHUB_TOKEN`, so dropping
    the persisted credential is safe.
  - In release/publishing workflows, set `cache: false` on `actions/setup-go`
    so a poisoned build cache cannot taint signed artifacts (`cache-poisoning`).
  - Keep all third-party actions pinned to a full commit SHA.
- **DevSkim false positives on protocol text:** emitted protocol strings can
  trip "suspicious comment" rules — e.g. the TAP `# TODO`/`# SKIP` directives
  in `render/tap.go`. Suppress inline with `// DevSkim: ignore <ID>` plus a
  one-line justification; do **not** alter the protocol output. Add the
  justification without the trigger word so the comment itself stays clean.

Go formatting, vet, and lint run through the Taskfile (see below); keep
`gofmt`/`golangci-lint`/`go vet` clean.

## Build, test, and verify

Everything runs through [Task](https://taskfile.dev) via `go tool task`
(no global installs — `task`, `ginkgo`, and `golangci-lint` are in `go.mod`'s
`tool` block). Key targets:

    go tool task build             # build bin/ginkgoleaf
    go tool task test              # run the suite (MODE=llm for errors-only)
    go tool task go:vet            # go vet
    go tool task lint              # golangci-lint
    go tool task go:vuln           # govulncheck
    go tool task go:gosum          # assert gomega does not leak into no-gomega consumers
    go tool task go:test:fixtures  # dogfood: render the Ginkgo fixtures
    go tool task ci             # the full gate CI runs — run before a PR

The repo uses a Go workspace (`go.work`). If your shell exports
`GOFLAGS=-mod=mod`, workspace builds fail with a `-mod` error — unset it
(`env -u GOFLAGS ...`) or rely on the Taskfile, which the CI does not override.

## Commits

Conventional Commits prefixes (`feat:`, `fix:`, `build:`, `ci:`, `docs:`,
`test:`, `chore:`) — the changelog is generated from them. Sign agent-authored
commits with a `Co-Authored-By:` trailer. Stage files by name; review
`git diff --staged` before committing. Do not commit changes to CI/CD
workflows, or anything outward-facing, without explicit user confirmation.
