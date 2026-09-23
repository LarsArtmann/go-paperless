package paperlesstest

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"slices"
	"testing"
	"time"

	paperless "github.com/larsartmann/go-paperless"
)

func TestUpdateDocumentPatchRoundTrip(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithDocuments(Document{
		Title:   "original",
		Content: []byte("%PDF-original"),
	}))
	client := newTestClient(t, server)
	ctx := context.Background()

	newTitle := "patched"
	newCreated := time.Date(2024, 12, 24, 0, 0, 0, 0, time.UTC)
	correspondent := 3
	documentType := 2

	err := client.UpdateDocument(ctx, 1, paperless.UpdateDocumentRequest{
		Title:           &newTitle,
		Created:         &newCreated,
		CorrespondentID: &correspondent,
		TagIDs:          []int{5, 6},
		DocumentTypeID:  &documentType,
		CustomFields:    []paperless.CustomFieldValue{{Field: 9, Value: "abc"}},
	})
	if err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}

	metas, err := client.ListDocumentMetas(ctx)
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 1 {
		t.Fatalf("metas = %d entries, want 1", len(metas))
	}

	meta := metas[0]
	if meta.Title != "patched" || meta.Correspondent != 3 || meta.DocumentTypeID != 2 {
		t.Errorf("patched meta = %+v, want title patched, correspondent 3, type 2", meta)
	}

	if !slices.Equal(meta.TagIDs, []int{5, 6}) {
		t.Errorf("patched tags = %v, want [5 6]", meta.TagIDs)
	}

	if len(meta.CustomFields) != 1 || meta.CustomFields[0].Field != 9 {
		t.Errorf("patched custom fields = %v, want field 9", meta.CustomFields)
	}

	patches := server.Patches()
	if len(patches) != 1 || patches[0].DocumentID != 1 {
		t.Fatalf("patches = %+v, want one patch against document 1", patches)
	}

	var recorded map[string]any
	if err := json.Unmarshal(patches[0].Body, &recorded); err != nil {
		t.Fatalf("decode recorded patch body: %v", err)
	}

	if _, ok := recorded["checksum"]; ok {
		t.Error("recorded patch carries a checksum field; the SDK must only patch metadata")
	}
}

func TestUpdateDocumentLeavesUntouchedFieldsAlone(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithDocuments(Document{
		Title:         "keep",
		Content:       []byte("%PDF-keep"),
		Correspondent: 7,
		TagIDs:        []int{1},
	}))
	client := newTestClient(t, server)

	newTitle := "only-title"
	if err := client.UpdateDocument(context.Background(), 1, paperless.UpdateDocumentRequest{
		Title: &newTitle,
	}); err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}

	stored := server.Documents()[0]
	if stored.Correspondent != 7 || !slices.Equal(stored.TagIDs, []int{1}) {
		t.Errorf("stored doc = %+v; a title-only patch must not touch correspondent/tags", stored)
	}
}

func TestDeleteDocumentRecordsAndRemoves(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithDocuments(
		Document{Title: "keep", Content: []byte("%PDF-keep")},
		Document{Title: "drop", Content: []byte("%PDF-drop")},
	))
	client := newTestClient(t, server)
	ctx := context.Background()

	if err := client.DeleteDocument(ctx, 2); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}

	if got := server.DeletedDocuments(); !slices.Equal(got, []int{2}) {
		t.Errorf("deleted documents = %v, want [2]", got)
	}

	metas, err := client.ListDocumentMetas(ctx)
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 1 || metas[0].Title != "keep" {
		t.Errorf("metas after delete = %+v, want only the kept document", metas)
	}
}

func TestDownloadDocumentServesStoredContent(t *testing.T) {
	t.Parallel()

	content := []byte("%PDF-download-me")

	server := NewServer(t, WithDocuments(Document{Title: "doc", Content: content}))
	client := newTestClient(t, server)

	downloaded, err := client.DownloadDocument(context.Background(), 1)
	if err != nil {
		t.Fatalf("DownloadDocument: %v", err)
	}

	if !bytes.Equal(downloaded, content) {
		t.Errorf("downloaded = %q, want %q", downloaded, content)
	}
}

func TestDetailRoutesAnswer404ForUnknownDocuments(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	client := newTestClient(t, server)
	ctx := context.Background()

	if err := client.DeleteDocument(ctx, 99); err == nil {
		t.Error("delete of unknown document = nil error, want a client rejection")
	}

	if err := client.UpdateDocument(ctx, 99, paperless.UpdateDocumentRequest{}); err == nil {
		t.Error("update of unknown document = nil error, want either the local empty-update rejection or a 404")
	}

	if _, err := client.DownloadDocument(ctx, 99); err == nil {
		t.Error("download of unknown document = nil error, want a client rejection")
	}
}
