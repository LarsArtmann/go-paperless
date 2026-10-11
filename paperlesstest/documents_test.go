package paperlesstest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/go-paperless"
)

func TestDocumentListServesFlatChecksums(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithChecksumShape(ChecksumFlat))
	server.addDocument(Document{Title: "one", Checksum: "checksum-one"})
	server.addDocument(Document{Title: "two", Checksum: "checksum-two"})

	checksums, err := newTestClient(t, server).ListDocumentChecksums(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentChecksums: %v", err)
	}

	if len(checksums) != 2 {
		t.Fatalf("checksums = %d entries, want 2", len(checksums))
	}

	for _, want := range []string{"checksum-one", "checksum-two"} {
		if _, ok := checksums[want]; !ok {
			t.Errorf("checksums missing %q", want)
		}
	}
}

func TestDocumentListServesVersionsChecksums(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithChecksumShape(ChecksumVersions))
	server.addDocument(Document{Title: "three", Checksum: "checksum-three"})

	client := newTestClient(t, server)

	checksums, err := client.ListDocumentChecksums(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentChecksums: %v", err)
	}

	if _, ok := checksums["checksum-three"]; !ok {
		t.Fatalf(
			"checksums missing %q (versions[] shape not resolved by the client)",
			"checksum-three",
		)
	}

	probe, err := client.ProbeCapabilities(context.Background())
	if err != nil {
		t.Fatalf("ProbeCapabilities: %v", err)
	}

	if !probe.VersionedChecksum || probe.FlatChecksum {
		t.Errorf("probe = %s (flat=%v versioned=%v), want versions[] only",
			probe.ChecksumShape(), probe.FlatChecksum, probe.VersionedChecksum)
	}
}

func TestDocumentListPaginatesBeyondOnePage(t *testing.T) {
	t.Parallel()

	server := NewServer(t)

	const documentCount = defaultPageSize + 7

	for i := range documentCount {
		server.addDocument(Document{Title: "doc", Checksum: checksumNumber(i)})
	}

	checksums, err := newTestClient(t, server).ListDocumentChecksums(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentChecksums: %v", err)
	}

	if len(checksums) != documentCount {
		t.Fatalf(
			"checksums = %d entries, want %d (pagination stopped early)",
			len(checksums),
			documentCount,
		)
	}
}

func TestDocumentListServesFullMetaShape(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	created := time.Date(2026, 9, 23, 10, 30, 0, 0, time.UTC)
	server.addDocument(Document{
		Title:          "invoice.pdf",
		Content:        []byte("%PDF-invoice"),
		Created:        created,
		Correspondent:  paperless.NewCorrespondentID(12),
		TagIDs:         []paperless.TagID{paperless.NewTagID(3), paperless.NewTagID(9)},
		DocumentTypeID: paperless.NewDocumentTypeID(5),
		CustomFields:   []CustomFieldValue{{Field: paperless.NewCustomFieldID(2), Value: "msg-42"}},
	})

	metas, err := newTestClient(t, server).ListDocumentMetas(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 1 {
		t.Fatalf("metas = %d entries, want 1", len(metas))
	}

	meta := metas[0]
	if meta.ID.Get() != 1 || meta.Title != "invoice.pdf" || meta.Correspondent.Get() != 12 {
		t.Errorf("meta = %+v, want id 1, title invoice.pdf, correspondent 12", meta)
	}

	if !meta.Created.Equal(created) {
		t.Errorf("created = %v, want %v", meta.Created, created)
	}

	if !slices.Equal(
		meta.TagIDs,
		[]paperless.TagID{paperless.NewTagID(3), paperless.NewTagID(9)},
	) ||
		meta.DocumentTypeID.Get() != 5 {
		t.Errorf("tags/type = %v/%d, want [3 9]/5", meta.TagIDs, meta.DocumentTypeID)
	}

	if len(meta.CustomFields) != 1 || meta.CustomFields[0].Field.Get() != 2 ||
		meta.CustomFields[0].Value != "msg-42" {
		t.Errorf("custom fields = %v, want [{2 msg-42}]", meta.CustomFields)
	}

	if meta.Checksum != ChecksumOf([]byte("%PDF-invoice")) {
		t.Errorf("checksum = %q, want derived SHA-256 of content", meta.Checksum)
	}
}

func TestPingTraversesDocumentList(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	server.addDocument(Document{Title: "one", Checksum: "checksum-one"})

	if err := newTestClient(t, server).Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

// checksumNumber derives a stable, unique-per-index checksum for bulk
// fixture documents.
func checksumNumber(i int) string {
	return ChecksumOf([]byte{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)})
}

func TestAddAndEditDocumentsAtRuntime(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	client := newTestClient(t, server)
	ctx := context.Background()

	stored := server.AddDocument(Document{Content: []byte("%PDF-runtime")})

	if stored.ID.Get() != 1 || stored.Checksum != ChecksumOf([]byte("%PDF-runtime")) {
		t.Fatalf("stored = %+v, want id 1 with derived checksum", stored)
	}

	aged := server.EditDocument(paperless.NewDocumentID(1), func(doc *Document) {
		doc.Title = ""
		doc.Correspondent = paperless.CorrespondentID{}
	})

	if !aged {
		t.Fatal("EditDocument = false for document 1")
	}

	if server.EditDocument(paperless.NewDocumentID(99), func(doc *Document) {}) {
		t.Error("EditDocument = true for unknown document 99")
	}

	metas, err := client.ListDocumentMetas(ctx)
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 1 || metas[0].Title != "" {
		t.Errorf("metas after edit = %+v, want the aged fixture", metas)
	}
}

func TestPageBoundsClampsOverflowingPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		total     int
		wantStart int
		wantEnd   int
	}{
		{name: "first page of a small collection", query: "", total: 5, wantStart: 0, wantEnd: 5},
		{
			name:      "page beyond the collection clamps to an empty page",
			query:     "page=2",
			total:     5,
			wantStart: 5,
			wantEnd:   5,
		},
		{
			name:      "page whose offset overflows int64 clamps to an empty page",
			query:     "page=100000000000000000",
			total:     5,
			wantStart: 5,
			wantEnd:   5,
		},
		{
			name:      "max int64 page clamps to an empty page",
			query:     "page=9223372036854775807",
			total:     5,
			wantStart: 5,
			wantEnd:   5,
		},
		{
			name:      "huge page size serves the whole collection",
			query:     "page_size=9223372036854775807",
			total:     5,
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:      "huge page and page size clamp to an empty page",
			query:     "page=9223372036854775807&page_size=2",
			total:     5,
			wantStart: 5,
			wantEnd:   5,
		},
	}

	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, "/api/documents/?"+test.query, nil)

		start, end := pageBounds(request, test.total)
		if start != test.wantStart || end != test.wantEnd {
			t.Errorf(
				"pageBounds(%q, %d) = (%d, %d), want (%d, %d)",
				test.query, test.total, start, end, test.wantStart, test.wantEnd,
			)
		}
	}
}
