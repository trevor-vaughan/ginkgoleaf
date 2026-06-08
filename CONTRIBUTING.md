# Contributing to ginkgoleaf

Thanks for your interest in improving ginkgoleaf.

## Prerequisites

- Go 1.25 or newer (1.26 recommended). The repo uses a Go workspace
  (`go.work`) that includes the root module and the fixture modules.
- No global tool installs needed: `task`, `ginkgo`, and `golangci-lint`
  are declared in `go.mod`'s `tool` block and run via `go tool <name>`.

## Development workflow

    go tool task build             # build bin/ginkgoleaf
    go tool task test              # run the suite, render the tree report
    go tool task go:test:race      # run with the race detector
    go tool task go:test:fixtures  # dogfood: render the Ginkgo fixtures
    go tool task lint              # golangci-lint
    go tool task go:vet            # go vet
    go tool task go:vuln           # govulncheck
    go tool task check             # the full gate CI runs

Run `go tool task check` before opening a pull request.

## Conventions

- Follow existing code style; `go tool task lint` and `go:vet` must pass.
- Golden files: change render output deliberately, then regenerate with
  `go tool task go:golden:update` and review the diff before committing.
- Commit messages use [Conventional Commits](https://www.conventionalcommits.org/)
  prefixes (`feat:`, `fix:`, `build:`, `ci:`, `docs:`, `test:`, `chore:`) —
  the release changelog is generated from them.
- Add tests (including negative cases) for new behavior.

## Reporting bugs

Open an issue with the ginkgoleaf version (`ginkgoleaf --version`), the
input JSON if possible, and the command you ran.
