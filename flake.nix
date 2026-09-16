{
  description = "go-paperless — Paperless-ngx REST client SDK for Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      treefmt-nix,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      # x86_64-darwin is omitted deliberately: nixpkgs 26.11 dropped it, and
      # a system entry that cannot evaluate breaks the flake for everyone.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      imports = [
        treefmt-nix.flakeModule
      ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          goPkg = pkgs.go_1_27;
          # The `simd` experiment only enables the `simd` stdlib package —
          # nothing in this module imports it, so it carries no weight here.
          goExperiment = "jsonv2";
          # Release version — bump together with CHANGELOG.md and the git tag.
          version = "0.3.1";

          # buildGoModule fetches Go modules into a fixed-output derivation
          # (network access there) and materialises them as vendor/ inside the
          # sandboxed build, so no check ever needs DNS.
          goModule = pkgs.buildGoModule.override { go = goPkg; };

          goModuleArgs = {
            inherit version;
            pname = "go-paperless";
            src = self;
            vendorHash = "sha256-mk/dJNFBbkK/hv6kYsOAiesEoMHkMqeNiVwnIXG5nG4=";
            env = {
              CGO_ENABLED = "0";
              GOEXPERIMENT = goExperiment;
            };
            meta = {
              description = "Paperless-ngx REST client SDK for Go";
              homepage = "https://github.com/LarsArtmann/go-paperless";
              license = lib.licenses.mit;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              platforms = lib.platforms.unix;
              # No mainProgram: library-only marker output, nothing to run.
            };
          };

          # Library-only module: there are no binaries to install, so the
          # build compiles every package and leaves a marker output. Shared
          # by checks.build and packages.default so bare `nix build` works.
          moduleBuild = goModule (
            goModuleArgs
            // {
              buildPhase = ''
                runHook preBuild
                go build ./...
                runHook postBuild
              '';
              installPhase = ''touch "$out"'';
            }
          );

          # goimports shells out to a `go` binary (gotools appends one to
          # PATH); pin it to the module toolchain so a newer `go` directive
          # in go.mod can never trigger an in-sandbox toolchain download.
          gotoolsForModule = pkgs.gotools.override { go = goPkg; };

          mkApp = name: description: runtimeInputs: text: {
            type = "app";
            meta.description = description;
            program = "${
              pkgs.writeShellApplication {
                inherit name runtimeInputs;
                text = ''
                  export GOEXPERIMENT=${goExperiment}
                  ${text}
                '';
              }
            }/bin/${name}";
          };
        in
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              gofumpt.enable = true;
              goimports = {
                enable = true;
                package = gotoolsForModule;
              };
              golines.enable = true;
              nixfmt.enable = true;
            };
          };

          devShells = {
            default = pkgs.mkShellNoCC {
              # Tools BuildFlow probes inside `nix develop` (its preflight
              # warns per-tool when missing); all nixpkgs, so the shell
              # stays hermetic.
              packages = [
                goPkg
                pkgs.dprint
                pkgs.go-licenses
                pkgs.golangci-lint
                pkgs.gotools
                pkgs.govulncheck
                pkgs.lychee
                pkgs.trash-cli
                pkgs.vulnix
              ];

              env = {
                GOEXPERIMENT = goExperiment;
                GOTOOLCHAIN = "local";
              };

              shellHook = ''
                echo "go-paperless dev shell — $(go version) (GOEXPERIMENT=$GOEXPERIMENT)"
              '';
            };

            ci = pkgs.mkShellNoCC {
              packages = [
                goPkg
                pkgs.golangci-lint
              ];
              env = {
                GOEXPERIMENT = goExperiment;
                GOTOOLCHAIN = "local";
              };
            };
          };

          packages = {
            go-paperless = moduleBuild;
            default = moduleBuild;
          };

          checks = {
            format = config.treefmt.build.check self;

            build = moduleBuild;

            # Race detector in CI-identical hermetic conditions: reuse the
            # vendored module build and force -race in its test phase.
            test-race = goModule (
              goModuleArgs
              // {
                pname = "go-paperless-race";
                env = goModuleArgs.env // {
                  CGO_ENABLED = "1";
                };
                checkFlags = [ "-race" ];
                installPhase = ''touch "$out"'';
              }
            );

            test = goModule (
              goModuleArgs
              // {
                doCheck = true;
                buildPhase = ''
                  runHook preBuild
                  go build ./...
                  runHook postBuild
                '';
                checkPhase = ''
                  runHook preCheck
                  go test ./... -count=1
                  runHook postCheck
                '';
                installPhase = ''touch "$out"'';
              }
            );

            lint = goModule (
              goModuleArgs
              // {
                nativeBuildInputs = [ pkgs.golangci-lint ];
                buildPhase = ''
                  runHook preBuild
                  export HOME=$TMPDIR
                  export GOLANGCI_LINT_CACHE=$TMPDIR/golangci-lint-cache
                  golangci-lint run ./...
                  runHook postBuild
                '';
                installPhase = ''touch "$out"'';
              }
            );

            # The integration scaffold must never rot uncompiled: vet+build
            # with the `integration` build tag type-checks
            # integration_test.go without a live server. The live run is
            # opt-in via `nix run .#integration` (needs a real
            # Paperless-ngx) — deliberately NOT in `nix flake check`.
            integration-vet = goModule (
              goModuleArgs
              // {
                buildPhase = ''
                  runHook preBuild
                  go vet -tags integration ./...
                  go build -tags integration ./...
                  runHook postBuild
                '';
                installPhase = ''touch "$out"'';
              }
            );
          };

          apps = {
            check =
              mkApp "check" "Run every flake check (build, test, lint, format) — CI equivalent" [ pkgs.nix ]
                ''
                  exec nix flake check "$@"
                '';

            test = mkApp "test" "Run the Go test suite" [ goPkg ] ''
              go test ./... -count=1 "$@"
            '';

            test-race = mkApp "test-race" "Run the Go test suite with the race detector" [ goPkg ] ''
              go test ./... -race -count=1 "$@"
            '';

            build = mkApp "build" "Compile all packages" [ goPkg ] ''
              go build ./...
            '';

            vet = mkApp "vet" "Run go vet over all packages" [ goPkg ] ''
              go vet ./...
            '';

            lint =
              mkApp "lint" "Run golangci-lint over all packages"
                [
                  goPkg
                  pkgs.golangci-lint
                ]
                ''
                  golangci-lint run ./...
                '';

            coverage = mkApp "coverage" "Run tests with coverage report" [ goPkg ] ''
              go test ./... -coverprofile=coverage.out -covermode=atomic "$@"
              go tool cover -func=coverage.out
            '';

            fmt =
              mkApp "fmt" "Format the tree via treefmt (gofumpt, goimports, golines, nixfmt)"
                [ config.treefmt.build.wrapper ]
                ''
                  treefmt "$@"
                '';

            clean =
              mkApp "clean" "Remove coverage output and Go test cache"
                [
                  goPkg
                  pkgs.trash-cli
                ]
                ''
                  trash-put coverage.out 2>/dev/null || true
                  go clean -testcache
                '';

            integration =
              mkApp "integration"
                "Run real-server integration tests (needs PAPERLESS_INTEGRATION_URL and PAPERLESS_INTEGRATION_TOKEN)"
                [ goPkg ]
                ''
                  go test -tags integration -count=1 -v ./...
                '';

            fuzz =
              mkApp "fuzz"
                "Run every fuzz target for a per-target duration (default 30s, e.g. nix run .#fuzz -- 5m)"
                [ goPkg ]
                ''
                  duration="''${1:-30s}"
                  while read -r pkg target; do
                    [[ -n "$target" ]] || continue
                    echo "=== fuzz $target ($pkg, $duration) ==="
                    # </dev/null: go test must not consume the loop's stdin.
                    go test -run "^''${target}$" -fuzz "^''${target}$" -fuzztime "$duration" "$pkg" < /dev/null
                  done < <(go test -list '^Fuzz' ./... | awk '/^Fuzz/ {names[n++]=` + "''$0" + `; next} /^ok/ {for (i=0; i<n; i++) print ` + "''$2" + `, names[i]; n=0}')
                '';

            release-verify =
              mkApp "release-verify"
                "Post-release ritual: tag/CHANGELOG/flake agreement, clean pushed tree, green CI, pkg.go.dev indexing, race tests"
                [
                  pkgs.bash
                  pkgs.coreutils
                  pkgs.curl
                  pkgs.git
                  pkgs.gh
                  goPkg
                ]
                ''
                  set -euo pipefail
                  version="''${1:-}"
                  if [[ -z "$version" ]]; then
                    echo "usage: nix run .#release-verify -- vX.Y.Z" >&2
                    exit 1
                  fi
                  version_digits="''${version#v}"

                  echo "== tag exists =="
                  git fetch --tags --quiet
                  git rev-parse -q --verify "refs/tags/$version" >/dev/null \
                    || { echo "FAIL: tag $version does not exist"; exit 1; }
                  echo "ok: $version"

                  echo "== CHANGELOG and flake.nix agree with the tag =="
                  grep -q "^## \[$version_digits\]" CHANGELOG.md \
                    || { echo "FAIL: CHANGELOG.md has no $version_digits heading"; exit 1; }
                  grep -q "version = \"$version_digits\"" flake.nix \
                    || { echo "FAIL: flake.nix version is not $version_digits"; exit 1; }
                  echo "ok: CHANGELOG + flake.nix = $version_digits"

                  echo "== working tree clean and pushed =="
                  test -z "$(git status --porcelain)" \
                    || { echo "FAIL: dirty working tree"; git status --short; exit 1; }
                  git fetch --quiet
                  ahead=$(git rev-list --count "origin/main..HEAD")
                  behind=$(git rev-list --count "HEAD..origin/main")
                  test "$ahead" -eq 0 -a "$behind" -eq 0 \
                    || { echo "FAIL: main is ahead=$ahead behind=$behind of origin"; exit 1; }
                  echo "ok: main == origin/main"

                  echo "== CI green on origin/main =="
                  gh run list --branch main --limit 1 --json status,conclusion \
                    --jq '.[0] | select(.status == "completed" and .conclusion == "success")' >/dev/null \
                    || { echo "FAIL: latest CI run on main is not green"; exit 1; }
                  echo "ok: CI green"

                  echo "== pkg.go.dev serves the version =="
                  curl -fsSL -o /dev/null \
                    "https://pkg.go.dev/github.com/larsartmann/go-paperless@$version" \
                    || { echo "FAIL: pkg.go.dev has not indexed $version yet"; exit 1; }
                  echo "ok: pkg.go.dev indexed $version"

                  echo "== race detector over the tagged tree =="
                  git worktree add "$TMPDIR/go-paperless-release-verify" "$version" >/dev/null
                  trap 'git worktree remove --force "$TMPDIR/go-paperless-release-verify"' EXIT
                  ( cd "$TMPDIR/go-paperless-release-verify" \
                    && CGO_ENABLED=1 go test -race -count=1 ./... )
                  echo "ALL GREEN: $version is released, indexed, and race-clean"
                '';
          };
        };
    };
}
