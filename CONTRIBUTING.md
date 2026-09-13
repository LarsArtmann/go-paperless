# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

Use the Nix flake for everything — bare `go` invocations fail on this module
(it uses `encoding/json/v2` and needs `GOEXPERIMENT=jsonv2`, see AGENTS.md):

    nix develop          # dev shell with the right environment
    nix run .#build      # build
    nix run .#test       # tests
    nix run .#test-race  # tests with the race detector
    nix run .#lint       # golangci-lint
    nix fmt              # format
    nix flake check      # all checks (what CI runs)

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
