<!--
Thanks for contributing! Keep PRs focused; one logical change per PR.
CI runs `nix flake check` (build, test, race, lint, format), govulncheck,
and gosec — all must pass before review.
-->

## What

<!-- What problem does this solve? Link the issue if there is one. -->

## Why

<!-- The user-facing or maintenance benefit. A reviewer reading only this
section should understand the impact without opening the diff. -->

## Verification

<!-- How you proved it works: tests added/updated, commands run
(e.g. `nix flake check`). Public API changes must update README,
FEATURES.md, and CHANGELOG.md under [Unreleased]. -->

- [ ] `nix fmt` clean
- [ ] `nix flake check` green
- [ ] Docs + CHANGELOG updated (if user-visible)
