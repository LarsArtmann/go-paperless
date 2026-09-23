# paperlesstest

A stateful in-memory fake of the Paperless-ngx REST API (API version 10)
for consumer tests. Standard-library-only: it never imports `paperless` or
any third-party module, so adding it to your tests keeps your `go.mod`
graph untouched.

Every route the fake serves is exercised round-trip through the real
`paperless.Client` inside this module, and an endpoint-coverage test fails
the build when the SDK grows a route the fake does not speak — the fake
cannot drift from what the SDK decodes.

## Quick start

```go
func TestSyncUploadsDocuments(t *testing.T) {
    srv := paperlesstest.NewServer(t)

    client, err := paperless.New(srv.URL(), paperlesstest.DefaultToken)
    if err != nil {
        t.Fatal(err)
    }

    ctx := context.Background()

    taskID, err := client.Upload(ctx, paperless.UploadRequest{
        Filename: "invoice.pdf",
        Content:  []byte("%PDF-invoice"),
    })
    if err != nil {
        t.Fatal(err)
    }

    outcome, err := client.WaitForTask(ctx, taskID, time.Millisecond)
    if err != nil {
        t.Fatal(err)
    }

    if outcome.Status != paperless.TaskStatusSuccess {
        t.Fatalf("outcome = %+v, want success", outcome)
    }
}
```

Unexpected requests answer 404 and fail the test via `t.Errorf`, so a
missing route surfaces immediately instead of as a mysterious failure.

## Seeding state

```go
srv := paperlesstest.NewServer(t,
    paperlesstest.WithChecksumShape(paperlesstest.ChecksumVersions), // 3.x shape (default: flat)
    paperlesstest.WithDocuments(paperlesstest.Document{
        Title:   "existing",
        Content: []byte("%PDF-existing"), // checksum derived via SHA-256
    }),
    paperlesstest.WithTags("gmail"),
    paperlesstest.WithCorrespondents("Acme Corp"),
    paperlesstest.WithStoragePaths(paperlesstest.StoragePathFixture{
        Name: "Invoices", Path: "{created_year}/invoices",
    }),
)
```

Successful uploads take the natural consumption path: the document is
stored with its form metadata (title, created, correspondent, tags,
document type, custom fields) and a SHA-256 content checksum — so a
re-upload of the same bytes is refused as a duplicate, exactly like the
real server.

## Scripting tasks

Task outcomes resolve first-in-first-out from the script queue; once dry,
uploads take the natural path.

```go
srv := paperlesstest.NewServer(t,
    paperlesstest.WithTaskFailure("OCR exploded"),        // first upload fails
    paperlesstest.WithTaskPendingThenSuccess(2),          // second: 2 pending polls, then consumed
)
// mid-test: srv.ScriptTask(paperlesstest.TaskPlan{
//     Kind: paperlesstest.TaskPlanDuplicate, DocumentID: 42, InTrash: false,
// })
```

- `TaskPlanSuccess` — consumed; `DocumentID` points at a fixture, or zero
  consumes the upload naturally (storing it).
- `TaskPlanFailure` — real failure; surfaces as the client's `task_failed`
  error with `ErrorMessage`.
- `TaskPlanDuplicate` — refusal: failure status with a `duplicate_of`
  `result_data`; the client reports it as a non-error duplicate outcome.

## Faults

```go
srv.InjectFault(paperlesstest.Fault{
    Method:     "GET",
    PathPrefix: "/api/documents/",
    Times:      2,                        // fail twice, then behave
    Status:     http.StatusTooManyRequests,
    RetryAfter: "3",                      // seconds or HTTP-date
})
```

A `Body` override serves verbatim bytes (malformed JSON exercises the
decode-corruption paths). 429/503 with `Retry-After` produce the client's
`RetryAfterError` with the parsed hint.

## Auth and assertions

```go
srv := paperlesstest.NewServer(t, paperlesstest.WithToken("secret"))

// ... run the code under test ...

paperlesstest.RequireAuthorized(t, srv)  // every request used the token
paperlesstest.RequireUploadCount(t, srv, 3)
```

The fake records everything:

| Accessor            | Returns                                                     |
| ------------------- | ----------------------------------------------------------- |
| `srv.Requests()`    | Every request (method, path, query, headers, raw body)      |
| `srv.Uploads()`     | Parsed uploads (filename, content, all form fields)         |
| `srv.Documents()`   | Stored document fixtures                                    |
| `srv.Patches()`     | Recorded document PATCHes (ID + raw JSON payload)           |
| `srv.DeletedDocuments()` | IDs removed through the detail route                   |
| `srv.NoteCount(id)` | Notes stored on one document                                |

## Scope

Wire protocol only. Ledger, projection, and pipeline logic is consumer
domain and deliberately stays out. Endpoint coverage is enforced
mechanically: `coverage_test.go` requires the fake's `servedRoutes` to
match client.go's route constants in both directions.

## Gates

The package runs under the repository's normal gates: `nix run .#test`,
`nix run .#test-race`, `nix run .#lint`.
