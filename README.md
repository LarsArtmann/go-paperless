# go-paperless

[![CI](https://github.com/LarsArtmann/go-paperless/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/go-paperless/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-paperless.svg)](https://pkg.go.dev/github.com/larsartmann/go-paperless)
[![Go](https://img.shields.io/badge/Go-1.27%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)

Paperless-ngx REST client SDK for Go.

One client, two consumers: [InboxClean](https://github.com/LarsArtmann/InboxClean) and
[bank-sync](https://github.com/LarsArtmann/bank-sync) share this client instead of
maintaining divergent forks. Scope is deliberately **client only** — document sync
pipelines are domain-coupled and stay in the consuming repos.

## Features

- Upload documents (content-hash friendly metadata: tags, correspondents, document types, custom fields)
- `Ensure*` idempotent lookups: `EnsureTag`, `EnsureCorrespondent`, `EnsureDocumentType`, `EnsureCustomField`, `EnsureStoragePath` (tags self-heal legacy auto-matching; storage paths keep their existing directory template)
- Task polling (`GetTask`, `TaskOutcome`, `WaitForTask`) for Paperless' async consumption pipeline, including duplicate-refusal detection (`TaskOutcome.Duplicate`)
- Document management: list checksums/metadata, update metadata, download, delete
- Document notes: `ListDocumentNotes`, `AddDocumentNote`, `DeleteDocumentNote`
- Share links: `CreateShareLink` (server-generated slug), `ListShareLinks`, `DeleteShareLink`
- Saved views: `ListSavedViews`, `CreateSavedView`, `DeleteSavedView`
- Storage paths: `FindStoragePath`, `EnsureStoragePath`, `ListStoragePaths`
- Custom fields: `FindCustomField` (read-only lookup), `EnsureCustomField`
- Name resolution: `GetCorrespondentName`, `GetDocumentTypeName`
- Capability probing (`ProbeCapabilities`) for version differences
- Respectful retry: `RetryAfterError` carries `Retry-After` hints; opt-in automatic retries via `WithRetry(RetryPolicy)` (transient-only, bodies replay, hints override backoff)
- Observability hooks: `WithRequestHook` / `WithResponseHook` value snapshots
- Typed errors via [go-error-family](https://github.com/LarsArtmann/go-error-family)

## Installation

```bash
go get github.com/larsartmann/go-paperless
```

## Requirements

Go 1.27+. The module uses `encoding/json/v2`, which is the default there —
no `GOEXPERIMENT` is needed. (On a Go ≤ 1.26 toolchain json/v2 only exists
behind `GOEXPERIMENT=jsonv2`, and builds fail with "build constraints
exclude all Go files" without it.)

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

## Common tasks

Document notes:

```go
notes, err := client.ListDocumentNotes(ctx, docID)
if err != nil {
    panic(err)
}

updated, err := client.AddDocumentNote(ctx, docID, "reviewed 2026-09-16")
if err != nil {
    panic(err)
}

err = client.DeleteDocumentNote(ctx, docID, updated[len(updated)-1].ID)
```

Share links (public, unauthenticated download URLs — see SECURITY.md):

```go
inWeek := time.Now().AddDate(0, 0, 7)

link, err := client.CreateShareLink(ctx, paperless.CreateShareLinkRequest{
    DocumentID:  docID,
    FileVersion: paperless.ShareLinkFileVersionOriginal,
    Expiration:  &inWeek,
})
if err != nil {
    panic(err)
}

fmt.Println("share URL:", baseURL+"/share/"+link.Slug)
```

Saved views:

```go
viewID, err := client.CreateSavedView(ctx, paperless.CreateSavedViewRequest{
    Name:        "This month's invoices",
    SortField:   "created",
    SortReverse: true,
    FilterRules: []paperless.SavedViewFilterRule{
        {RuleType: paperless.SavedViewRuleTypeHasTagsAll, Value: "invoice"},
        {RuleType: paperless.SavedViewRuleTypeCreatedAfter, Value: "2026-09-01"},
    },
})
if err != nil {
    panic(err)
}
```

## Options

| Option                                 | Effect                                                               |
| -------------------------------------- | -------------------------------------------------------------------- |
| `WithHTTPClient(*http.Client)`         | Use a custom HTTP client (transport, proxies, timeouts)              |
| `WithTimeout(time.Duration)`           | Per-request timeout on the default client                            |
| `WithRetry(RetryPolicy)`               | Opt-in automatic retries for transient failures (default: fail fast) |
| `WithRequestHook(func(RequestInfo))`   | Observe every outgoing request (headers include the token — redact)  |
| `WithResponseHook(func(ResponseInfo))` | Observe every response (2xx full body; errors capped at 512 bytes)   |

## Retries

Retries are opt-in and cover transient failures only (network errors, 429,
5xx). Rejections (401/403, other 4xx) fail fast. Request bodies replay
byte-for-byte per attempt; a server `Retry-After` hint overrides the
exponential backoff.

```go
client, err := paperless.New(url, token, paperless.WithRetry(paperless.RetryPolicy{
    MaxAttempts: 4,
}))
```

## Lookup verbs

| Verb      | Contract                                                        |
| --------- | --------------------------------------------------------------- |
| `Get*`    | Read by ID; not found is an error or a `false` flag             |
| `Find*`   | Read-only exact-name lookup; returns `exists=false` when absent |
| `Ensure*` | Find-or-create; never mutates an existing object's config       |

## Development

```bash
nix develop          # dev shell (Go 1.27, GOEXPERIMENT=jsonv2,simd)
nix run .#check      # all checks (build, test, lint, format)
nix build            # build the client (packages.default)
nix run .#build      # build
nix run .#test       # tests
nix run .#test-race  # tests with race detector
nix run .#lint       # golangci-lint
nix fmt              # format
nix flake check      # all checks (CI equivalent)
```

## License

MIT — see [LICENSE](LICENSE).

For security reporting, see [SECURITY.md](SECURITY.md); to contribute, see
[CONTRIBUTING.md](CONTRIBUTING.md).
