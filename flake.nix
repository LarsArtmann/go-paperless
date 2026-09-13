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
    systems.url = "github:nix-systems/default";
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      treefmt-nix,
      systems,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import systems;

      imports = [
        treefmt-nix.flakeModule
      ];

      perSystem =
        {
          config,
          pkgs,
          ...
        }:
        let
          goPkg = pkgs.go_1_27;
          goExperiment = "jsonv2,simd";
          version = "0.1.0";

          # buildGoModule fetches Go modules into a fixed-output derivation
          # (network access there) and materialises them as vendor/ inside the
          # sandboxed build, so no check ever needs DNS.
          goModule = pkgs.buildGoModule.override { go = goPkg; };

          goModuleArgs = {
            inherit version;
            pname = "go-paperless";
            src = self;
            vendorHash = "sha256-MQ+cMXFKmVbiuu7nSVSHpFANEyTjNc1xKd0ixEXjA50=";
            env = {
              CGO_ENABLED = "0";
              GOEXPERIMENT = goExperiment;
            };
          };

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

          checks.format = config.treefmt.build.check self;

          devShells = {
            default = pkgs.mkShellNoCC {
              packages = [
                goPkg
                pkgs.golangci-lint
                pkgs.gotools
                pkgs.trash-cli
              ];

              env = {
                GOEXPERIMENT = goExperiment;
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
              };
            };
          };

          checks = {
            build = goModule (
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
          };
        };
    };
}
