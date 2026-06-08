# ginkgoleaf

A third-party output formatter for [Ginkgo v2](https://github.com/onsi/ginkgo) that
renders test results in eight human- and machine-friendly formats. Drop it into
your suite as a library, or post-process Ginkgo's JSON report from the CLI.

**Status:** Pre-release. APIs may still change.

----

> 🤖 LLM WARNING 🤖
>
> This project was written with LLM (AI) assistance.
>
> 🤖 LLM WARNING 🤖

----

----

## Formats

| Format     | Mode      | Use it for                                                                                                                                                                       |
|------------|-----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `tree`     | batch     | The default human format — box-drawing tree per suite with bold headings, colored glyphs, and a trailing Failures block.                                                         |
| `jest`     | streaming | jest-style with check/X glyphs, BDD nesting headers, inline failures, OSC 8 file:line links.                                                                                     |
| `markdown` | batch     | `<details>` blocks per top-level container; failing groups expanded by default.                                                                                                  |
| `github`   | batch     | GitHub Actions workflow commands (`::group::`, `::error file=...,line=...`), preceded by a two-line human-readable header (suite path + counts) that Actions logs as plain text. |
| `gitlab`   | batch     | GitLab CI `section_start`/`section_end` markers + ANSI verdicts.                                                                                                                 |
| `text`     | streaming | Plain ASCII tree, never ANSI.                                                                                                                                                    |
| `shell`    | streaming | Tab-separated `STATE\tDURATION_MS\tFILE:LINE\tDESC` records, preceded by a `#fields:` header and followed by a `#summary` totals line — grep/awk friendly.                        |
| `tap`      | batch     | TAP 14 with YAML diagnostics blocks.                                                                                                                                             |

Color resolution everywhere: `--color=auto` (default) colors a terminal **or a
pipe** — so `ginkgoleaf … | less -R` stays colored — but writes plain output to
a regular file redirect (`> out.txt`), so captured output carries no escapes. It
honours `NO_COLOR` / `CLICOLOR` (<https://no-color.org>); `CLICOLOR_FORCE` forces
color on. `--color=always` and `--color=never` force the decision outright.

----

## What gets rendered

**Spec hierarchy, not run order.** The structured formats (`tree`, `github`,
`gitlab`, `markdown`) render each container exactly once, gathering its specs,
no matter what order Ginkgo executed them. Reports produced with
`--randomize-all` read just as cleanly as in-order runs — a `Describe` never
appears twice.

**Setup and teardown nodes.** Suite-level setup/teardown that *passed* —
`BeforeSuite`, `AfterSuite`, their `Synchronized` variants, and suite-scoped
`DeferCleanup` — is omitted, so ginkgoleaf's spec counts match Ginkgo's own
totals rather than the raw report's node count. A setup node that *failed* is
always shown, labelled by type (e.g. `[BeforeSuite]`), so a broken
`BeforeSuite` stays visible in every format.

----

## Two ways to use it

### 1. In-suite library

Call `ginkgoleaf.Register` once at file scope in any `_test.go` file. The
renderer hooks Ginkgo's `ReportAfterSuite`, so `go test` (or `ginkgo`) produces
the formatted output at end-of-suite — Ginkgo's own dots run normally above it.

```go
package mysuite_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "github.com/trevor-vaughan/ginkgoleaf/pkg/ginkgoleaf"
)

var _ = ginkgoleaf.Register(ginkgoleaf.FormatTree)

func TestMySuite(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "My Suite")
}
```

Options:

```go
ginkgoleaf.Register(
    ginkgoleaf.FormatTree,
    ginkgoleaf.WithColor(ginkgoleaf.ColorAuto),   // ColorAuto | ColorAlways | ColorNever
    ginkgoleaf.WithWriter(os.Stdout),             // io.Discard during integration tests
)
```

`Register` returns `bool` (always `true`) to match Ginkgo's DSL idiom so it
fits at file scope. Misconfiguration — unknown format, double registration —
panics, which Ginkgo surfaces at suite-init.

### 2. Standalone CLI (post-processing)

If you'd rather decouple rendering from the test run (recommended for CI and
for users who don't want a runtime dep), have Ginkgo write a JSON report and
pipe it through the binary:

```sh
# Until the first tagged release, build from a clone instead — the
# module is not yet published, so `go install …@latest` will fail:
#   git clone https://github.com/trevor-vaughan/ginkgoleaf && cd ginkgoleaf
#   go build ./cmd/ginkgoleaf            # produces ./ginkgoleaf
# Once released:
go install github.com/trevor-vaughan/ginkgoleaf/cmd/ginkgoleaf@latest

ginkgo --succinct --json-report=ginkgo.json -r ./...
ginkgoleaf --format=tree --in=ginkgo.json
```

Or as a single pipeline:

```sh
ginkgo --succinct --json-report=/tmp/g.json -r ./...
ginkgoleaf --format=markdown --in=/tmp/g.json --out=report.md
```

Flags:

```text
--in    PATH | -    Ginkgo JSON report (default: stdin)
--out   PATH | -    output path; a file path writes there only (stdout stays silent), - is stdout (default)
--format FMT        tree | jest | markdown | github | gitlab | text | shell | tap (default: tree)
--color  MODE       auto | always | never (default: auto)
--exit-code         exit 1 if the report contains failures (default: off)
--version           print build info
```

Unknown `--format` or `--color` values are rejected with exit `2` and a message
listing the valid choices.

For an interactive read of a large `tree`/`jest` report, pipe to a pager. Color
survives by default (a pipe is colored under `--color=auto`); `-R` tells `less`
to render the ANSI escapes:

```sh
ginkgoleaf --format=tree --in=ginkgo.json | less -R
```

(Set `NO_COLOR=1`, or pass `--color=never`, if you'd rather page it plain.)

Exit codes: `0` rendered cleanly, `1` render error, `2` invalid input / unknown
format or color. By default the CLI does **not** re-derive a pass/fail exit code
from the report — that's Ginkgo's job — so a combined gate carries Ginkgo's code
through:

```sh
ginkgo -r ./...; rc=$?
ginkgoleaf --format=tree --in=ginkgo.json
exit $rc
```

Or opt into `--exit-code` to fold the verdict into one command — it adds "exit
`1` when the report contains failures" on top of the render:

```sh
ginkgo --json-report=ginkgo.json -r ./...
ginkgoleaf --exit-code --format=tree --in=ginkgo.json   # exits 1 if any spec failed
```

----

## Optional gomega integration

The main `ginkgoleaf` package does **not** import [gomega][gomega]. Suites that
use only `Fail()` (or any other assertion library) render normally — the
failure parser falls back to the raw message string and the renderer carries
on. Verified by the `fixtures/no-gomega/` module + `tools/check-no-gomega.sh`.

If you do use gomega and want richer matcher diffs (structural JSON/YAML diffs,
substring highlights, Eventually retry context), import the opt-in helper:

```go
import (
    . "github.com/onsi/gomega"

    leafg "github.com/trevor-vaughan/ginkgoleaf/gomega" // opt-in gomega helpers
)

// ...
leafg.Expect(actualThing).To(Equal(expectedThing))
```

`leafg.Expect` wraps `gomega.Expect` and stashes a structured `MatcherEvent`
via `ginkgo.AddReportEntry` on failure. Renderers prefer that clean-path data
when present, and fall back to a best-effort failure-text parser otherwise.

----

## Examples in this repo

This project eats its own dogfood. The places to look:

* **`fixtures/with-gomega/suite_test.go`** — minimal Ginkgo suite using gomega.
  Demonstrates how a downstream user would structure tests, and intentionally
  contains a passing, failing, pending, and skipped spec so you can see every
  glyph in the tree.
* **`fixtures/no-gomega/suite_test.go`** — same shape, but without gomega. The
  no-gomega hygiene check (`tools/check-no-gomega.sh`) ensures importing this
  package never pulls gomega in transitively.
* **`Taskfile.yml`** — the `test`, `test:fixtures`, and `test:race` targets
  show how to drive Ginkgo with `--json-report` and pipe through ginkgoleaf
  for the per-suite tree.
* **`render/render_suite_test.go`** — exemplar pattern for a BDD test
  package: external `_test` package, `RegisterFailHandler` + `RunSpecs`, no
  in-suite Register (because the Taskfile runs the CLI separately).

----

## Recommended task flow

Build/test/lint targets run through [Task][task] — a YAML-driven task
runner. `task` itself is declared in this repo's `go.mod` `tool` block
(alongside `ginkgo` and `golangci-lint`), so once you've run
`go mod download` you can invoke any target without a global install:

```sh
go tool task                       # list available tasks
go tool task test                  # run suites + render tree per package
go tool task test MODE=jest        # …or jest|markdown|github|gitlab|text|shell|tap
go tool task go:test:race          # ditto with the race detector
go tool task go:test:fixtures      # run fixtures specifically (with-gomega + no-gomega)
go tool task lint                  # golangci-lint via go tool — no $GOBIN clutter
go tool task go:golden:update      # regenerate renderer goldens after intended changes
go tool task check                 # tidy + vet + lint + race tests + gosum check + build
```

If you'd rather type `task` directly, either install Task globally per
the [installation guide][task-install] or alias it (`alias task='go tool task'`).
CI follows the same pattern — see `.github/workflows/ci.yml`.

[task]: https://taskfile.dev
[task-install]: https://taskfile.dev/installation/

----

## CI integration

For GitHub Actions, the cleanest pattern is two steps: run Ginkgo into a JSON
report, then post-process. Annotations land inline on the PR diff:

```yaml
- name: Run tests
  run: |
    go tool ginkgo --succinct --skip-package=fixtures -r --keep-going \
      --json-report=ginkgo.json ./...

- name: Render PR annotations
  if: always()
  run: go run ./cmd/ginkgoleaf --format=github --in=ginkgo.json

- name: Render PR summary
  if: always()
  run: |
    go run ./cmd/ginkgoleaf --format=markdown --in=ginkgo.json --out=summary.md
    cat summary.md >> $GITHUB_STEP_SUMMARY
```

`--keep-going` matters across a multi-suite run (`-r ./...`): without it Ginkgo
stops at the first failing suite, and the report records the rest as
*"Suite did not run"* — ginkgoleaf then renders those as empty `no specs`
suites. With it, every suite runs and lands in the report, so the annotations
and summary are complete.

GitLab is symmetric — swap `--format=github` for `--format=gitlab` and remove
the step-summary step.

----

## Verifying releases

Each tagged release publishes, alongside the archives and `checksums.txt`:

- **SBOMs** — one SPDX-JSON `*.sbom.json` per archive (generated by syft).
- **A cosign signature** — `checksums.txt.sigstore.json`, a keyless Sigstore
  bundle covering the checksum file (and thus every artifact it lists).
- **Build provenance** — a GitHub-native SLSA attestation for each archive.

Verify the checksum signature (requires [cosign](https://github.com/sigstore/cosign)):

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp 'https://github.com/trevor-vaughan/ginkgoleaf/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```

Verify build provenance (requires the [`gh`](https://cli.github.com) CLI):

```bash
gh attestation verify ginkgoleaf_<version>_linux_amd64.tar.gz \
  --repo trevor-vaughan/ginkgoleaf
```

## License

MIT — see `LICENSE`.

[gomega]: https://github.com/onsi/gomega
