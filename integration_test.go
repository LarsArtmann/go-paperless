//go:build integration

// Integration tests against a real Paperless-ngx server. They are excluded
// from the default build (the `integration` tag gates them) and from
// `nix flake check`; run them explicitly with:
//
//	go test -tags integration ./...
//
// Required environment: PAPERLESS_INTEGRATION_URL (e.g.
// http://localhost:8000) and PAPERLESS_INTEGRATION_TOKEN (an API token from
// the Paperless web UI). Without both, every test in this file fails fast —
// the build tag already keeps them out of normal runs, so a silent skip
// would only hide a misconfigured invocation.
package paperless_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/larsartmann/go-paperless"
)

func integrationClient(t *testing.T) *paperless.Client {
	t.Helper()

	baseURL := os.Getenv("PAPERLESS_INTEGRATION_URL")
	token := os.Getenv("PAPERLESS_INTEGRATION_TOKEN")

	if baseURL == "" || token == "" {
		t.Fatalf(
			"integration tests need PAPERLESS_INTEGRATION_URL and PAPERLESS_INTEGRATION_TOKEN set",
		)
	}

	client, err := paperless.New(baseURL, token)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return client
}

func TestIntegrationPing(t *testing.T) {
	client := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestIntegrationListChecksums(t *testing.T) {
	client := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	checksums, err := client.ListDocumentChecksums(ctx)
	if err != nil {
		t.Fatalf("ListDocumentChecksums: %v", err)
	}

	t.Logf("server reports %d checksums", len(checksums))
}

func TestIntegrationListMetadatas(t *testing.T) {
	client := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	metas, err := client.ListDocumentMetas(ctx)
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	for _, meta := range metas {
		if meta.ID == 0 {
			t.Fatal("document meta with zero ID")
		}
	}

	t.Logf("server reports %d document metas", len(metas))
}

func TestIntegrationShareLinksRoundTrip(t *testing.T) {
	client := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	links, err := client.ListShareLinks(ctx)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}

	t.Logf("server reports %d share links", len(links))
}

func TestIntegrationSavedViewsRoundTrip(t *testing.T) {
	client := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	views, err := client.ListSavedViews(ctx)
	if err != nil {
		t.Fatalf("ListSavedViews: %v", err)
	}

	t.Logf("server reports %d saved views", len(views))
}
