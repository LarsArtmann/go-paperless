package paperlesstest

import (
	"context"
	"slices"
	"testing"
	"time"
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
		t.Fatalf("checksums missing %q (versions[] shape not resolved by the client)", "checksum-three")
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
		t.Fatalf("checksums = %d entries, want %d (pagination stopped early)", len(checksums), documentCount)
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
		Correspondent:  12,
		TagIDs:         []int{3, 9},
		DocumentTypeID: 5,
		CustomFields:   []CustomFieldValue{{Field: 2, Value: "msg-42"}},
	})

	metas, err := newTestClient(t, server).ListDocumentMetas(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 1 {
		t.Fatalf("metas = %d entries, want 1", len(metas))
	}

	meta := metas[0]
	if meta.ID != 1 || meta.Title != "invoice.pdf" || meta.Correspondent != 12 {
		t.Errorf("meta = %+v, want id 1, title invoice.pdf, correspondent 12", meta)
	}

	if !meta.Created.Equal(created) {
		t.Errorf("created = %v, want %v", meta.Created, created)
	}

	if !slices.Equal(meta.TagIDs, []int{3, 9}) || meta.DocumentTypeID != 5 {
		t.Errorf("tags/type = %v/%d, want [3 9]/5", meta.TagIDs, meta.DocumentTypeID)
	}

	if len(meta.CustomFields) != 1 || meta.CustomFields[0].Field != 2 || meta.CustomFields[0].Value != "msg-42" {
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
