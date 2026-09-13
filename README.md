# go-paperless

Paperless-ngx REST client SDK for Go.

One client, two consumers: [InboxClean](https://github.com/LarsArtmann/InboxClean) and
[bank-sync](https://github.com/LarsArtmann/bank-sync) share this client instead of
maintaining divergent forks. Scope is deliberately **client only** — document sync
pipelines are domain-coupled and stay in the consuming repos.

## Features

- Upload documents (content-hash friendly metadata: tags, correspondents, document types, custom fields)
- `Ensure*` idempotent lookups: `EnsureTag`, `EnsureCorrespondent`, `EnsureDocumentType`, `EnsureCustomField` (tags self-heal legacy auto-matching)
- Task polling (`GetTask`, `TaskOutcome`) for Paperless' async consumption pipeline, including duplicate-refusal detection
- Document management: list checksums/metadata, update metadata, download, delete
- Name resolution: `GetCorrespondentName`, `GetDocumentTypeName`
- Capability probing (`ProbeCapabilities`) for version differences
- Respectful retry: `RetryAfterError` carries `Retry-After` hints
- Typed errors via [go-error-family](https://github.com/LarsArtmann/go-error-family)

## Installation

```bash
go get github.com/larsartmann/go-paperless
```

## Requirements

Go 1.26+. The module uses `encoding/json/v2`, so **building anything that
imports it — including your own project — needs `GOEXPERIMENT=jsonv2`** in
the environment.

## Getting started

```go
package main

import (
    "context"
    "fmt"

    "github.com/larsartmann/go-paperless"
)

func main() {
    client, err := paperless.New("http://paperless.local:8000", "my-token")
    if err != nil {
        panic(err)
    }
    ctx := context.Background()
    if err := client.Ping(ctx); err != nil {
        panic(err)
    }
    tagID, err := client.EnsureTag(ctx, "inboxclean")
    if err != nil {
        panic(err)
    }
    fmt.Println(tagID)
}
```

## Options

| Option                         | Effect                                                  |
| ------------------------------ | ------------------------------------------------------- |
| `WithHTTPClient(*http.Client)` | Use a custom HTTP client (transport, proxies, timeouts) |
| `WithTimeout(time.Duration)`   | Per-request timeout on the default client               |

## Development

```bash
nix develop          # dev shell (Go 1.26, GOEXPERIMENT=jsonv2)
nix run .#build      # build
nix run .#test       # tests
nix run .#test-race  # tests with race detector
nix run .#lint       # golangci-lint
nix fmt              # format
nix flake check      # all checks (CI equivalent)
```

## License

MIT — see [LICENSE](LICENSE).
