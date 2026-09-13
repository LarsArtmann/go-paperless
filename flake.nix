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
          goPkg = pkgs.go_1_26;
          goExperiment = "jsonv2,goroutineleakprofile,simd";

          mkApp = name: runtimeInputs: text: {
            type = "app";
            program = "${
              pkgs.writeShellApplication {
                inherit name runtimeInputs;
                text = ''
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
              goimports.enable = true;
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
            build = pkgs.runCommand "go-paperless-build" { nativeBuildInputs = [ goPkg ]; } ''
              export HOME=$TMPDIR
              export CGO_ENABLED=0
              export GOEXPERIMENT=${goExperiment}
              cp -r ${./.} src && chmod -R u+w src && cd src
              ${goPkg}/bin/go build ./...
              touch $out
            '';

            build-standalone =
              pkgs.runCommand "go-paperless-build-standalone" { nativeBuildInputs = [ goPkg ]; }
                ''
                  export HOME=$TMPDIR
                  export CGO_ENABLED=0
                  export GOEXPERIMENT=${goExperiment}
                  export GOWORK=off
                  cp -r ${./.} src && chmod -R u+w src && cd src
                  ${goPkg}/bin/go build ./...
                  touch $out
                '';

            lint =
              pkgs.runCommand "go-paperless-lint"
                {
                  nativeBuildInputs = [
                    goPkg
                    pkgs.golangci-lint
                  ];
                }
                ''
                  export HOME=$TMPDIR
                  export CGO_ENABLED=0
                  export GOEXPERIMENT=${goExperiment}
                  cp -r ${./.} src && chmod -R u+w src && cd src
                  ${pkgs.golangci-lint}/bin/golangci-lint run ./...
                  touch $out
                '';
          };

          apps = {
            test = mkApp "test" [ goPkg ] ''
              go test ./... -count=1 "$@"
            '';

            test-race = mkApp "test-race" [ goPkg ] ''
              go test ./... -race -count=1 "$@"
            '';

            build = mkApp "build" [ goPkg ] ''
              go build ./...
            '';

            vet = mkApp "vet" [ goPkg ] ''
              go vet ./...
            '';

            lint = mkApp "lint" [ pkgs.golangci-lint ] ''
              golangci-lint run ./...
            '';

            coverage = mkApp "coverage" [ goPkg ] ''
              go test ./... -coverprofile=coverage.out -covermode=atomic "$@"
              go tool cover -func=coverage.out
            '';

            fmt = mkApp "fmt" [ config.treefmt.build.programs ] ''
              treefmt "$@"
            '';

            clean = mkApp "clean" [ goPkg pkgs.trash-cli ] ''
              trash-put coverage.out 2>/dev/null || true
              go clean -testcache
            '';
          };
        };
    };
}
