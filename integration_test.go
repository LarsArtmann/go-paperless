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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
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

// TestIntegrationUploadReconcile exercises the loop the SDK exists for:
// upload a unique document, wait for asynchronous consumption to finish,
// then prove the stored checksum appears in a full server listing. The
// document is deleted in cleanup so repeated runs stay idempotent.
func TestIntegrationUploadReconcile(t *testing.T) {
	client := integrationClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// A timestamp makes the bytes (and therefore the server-side SHA-256
	// checksum) unique across runs.
	content := []byte(fmt.Sprintf(
		"go-paperless integration %s\ndocument for upload/wait/reconcile\n",
		time.Now().UTC().Format(time.RFC3339Nano),
	))

	tagID, err := client.EnsureTag(ctx, "go-paperless-integration")
	if err != nil {
		t.Fatalf("EnsureTag: %v", err)
	}

	taskID, err := client.Upload(ctx, paperless.UploadRequest{
		Filename: fmt.Sprintf("go-paperless-integration-%d.txt", time.Now().UnixNano()),
		Content:  content,
		Title:    "go-paperless integration upload",
		TagIDs:   []int{tagID},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	outcome, err := client.WaitForTask(ctx, taskID, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForTask(%s): %v", taskID, err)
	}

	if _, _, refused := outcome.Duplicate(); refused {
		t.Fatalf("upload refused as duplicate of %d — content collision?", outcome.DocumentID)
	}

	if outcome.DocumentID == 0 {
		t.Fatalf("task %s consumed without a document ID: %+v", taskID, outcome)
	}

	documentID := int(outcome.DocumentID)

	t.Logf("consumed as document %d", documentID)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()

		if err := client.DeleteDocument(cleanupCtx, documentID); err != nil {
			t.Errorf("cleanup DeleteDocument(%d): %v", documentID, err)
		}
	})

	// The server computes the checksum over the stored file; the listing
	// must carry it after consumption.
	sha := sha256.Sum256(content)
	wantChecksum := hex.EncodeToString(sha[:])

	checksums, err := client.ListDocumentChecksums(ctx)
	if err != nil {
		t.Fatalf("ListDocumentChecksums: %v", err)
	}

	if _, found := checksums[wantChecksum]; !found {
		metas, metaErr := client.ListDocumentMetas(ctx)
		if metaErr != nil {
			t.Fatalf("document %d checksum %q missing from %d listed checksums (meta listing also failed: %v)",
				documentID, wantChecksum, len(checksums), metaErr)
		}

		for _, meta := range metas {
			if meta.ID == documentID {
				t.Fatalf("document %d listed with checksum %q, want %q (checksum shapes differ?)",
					meta.ID, meta.Checksum, wantChecksum)
			}
		}

		t.Fatalf("document %d vanished from the listing entirely", documentID)
	}

	metas, err := client.ListDocumentMetas(ctx)
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	for _, meta := range metas {
		if meta.ID != documentID {
			continue
		}

		if meta.Title != "go-paperless integration upload" {
			t.Errorf("title = %q", meta.Title)
		}

		if !slices.Contains(meta.TagIDs, tagID) {
			t.Errorf("tag IDs = %v, want %d among them", meta.TagIDs, tagID)
		}

		return
	}

	t.Fatalf("document %d missing from the meta listing", documentID)
}
