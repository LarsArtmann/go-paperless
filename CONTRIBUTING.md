# Contributing

Thanks for your interest in contributing!

## Picking Work

Open work lives in [TODO_LIST.md](TODO_LIST.md) (short-term, ranked by
impact); long-term ideas are in [ROADMAP.md](ROADMAP.md). Grab an item from
there, or file an issue first for anything larger.

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes (tests for new behavior, `CHANGELOG.md` entry)
4. Submit a pull request — CI runs `nix flake check` (build, tests, lint,
   format) plus `govulncheck` and `gosec`

## Development Setup

Use the Nix flake for everything — bare `go` invocations fail on Go ≤ 1.26
toolchains (the module uses `encoding/json/v2`, see AGENTS.md):

    nix develop          # dev shell with the right environment
    nix run .#check      # all checks (build, test, lint, format)
    nix run .#build      # build
    nix run .#test       # tests
    nix run .#test-race  # tests with the race detector
    nix run .#lint       # golangci-lint
    nix fmt              # format
    nix flake check      # all checks (what CI runs)

## Release Checklist (maintainers)

1. Curate `CHANGELOG.md`: `[Unreleased]` → new version header (Keep a
   Changelog categories; today's date).
2. Pre-tag verification: `grep '^replace' go.mod` empty, clean working
   tree, `nix flake check` green.
3. `git tag -a vX.Y.Z -m "..."` and push the tag.
4. Verify the module proxy resolves:
   `go list -m github.com/larsartmann/go-paperless@vX.Y.Z`.
5. Create the GitHub Release from the CHANGELOG section.

## Reporting Issues

Please use GitHub Issues to report bugs or request features. For security
vulnerabilities, see [SECURITY.md](SECURITY.md) instead.
