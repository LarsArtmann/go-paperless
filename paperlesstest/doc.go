// Package paperlesstest provides a stateful in-memory fake Paperless-ngx
// server for consumer tests.
//
// The fake speaks the same wire protocol as the real Paperless-ngx REST API
// (API version 10): DRF-style paginated envelopes, the post_document upload
// endpoint, consumption-task polling with result_data, named-entity
// management (tags, correspondents, document types, custom fields, storage
// paths), document detail PATCH/DELETE/download, notes, share links, saved
// views, and capability probing. It serves documents in both checksum
// shapes — the legacy flat field and the paperless-ngx 3.x versions[] array
// — selectable with WithChecksumShape.
//
// Every route the package serves is exercised round-trip through the real
// paperless.Client inside this module's tests, so the fake cannot drift
// from what the SDK expects. The endpoint-coverage test additionally fails
// when the SDK grows a route the fake does not speak.
//
// # Scope
//
// Wire protocol only: request/response shapes, status codes, and
// asynchronous task semantics. Consumer domain logic (ledgers, pipelines,
// projections) deliberately lives in the consumers, not here.
//
// # Dependencies
//
// Standard library only. The package never imports paperless or any third
// party module, keeping consumer go.mod graphs clean; round-trip tests
// against the real client live in this package's test files.
//
// # Quick start
//
//	srv := paperlesstest.NewServer(t)
//	client, err := paperless.New(srv.URL(), paperlesstest.DefaultToken)
//	// ... client.Upload / WaitForTask / ListDocumentChecksums against the fake
//
// Unexpected requests fail the test via t.Errorf and answer 404, so a fake
// that is missing a route the code under test calls surfaces immediately.
package paperlesstest
