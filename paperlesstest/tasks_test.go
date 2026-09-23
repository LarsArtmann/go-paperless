package paperlesstest

import (
	"context"
	"slices"
	"testing"
	"time"

	paperless "github.com/larsartmann/go-paperless"
)

func TestUploadReturnsTaskIDAndCapturesUpload(t *testing.T) {
	t.Parallel()

	server := NewServer(t)

	taskID, err := newTestClient(t, server).Upload(context.Background(), paperless.UploadRequest{
		Filename: "invoice.pdf",
		Content:  []byte("%PDF-invoice"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if taskID == "" {
		t.Fatal("Upload returned an empty task ID")
	}

	uploads := server.Uploads()
	if len(uploads) != 1 {
		t.Fatalf("captured uploads = %d, want 1", len(uploads))
	}

	upload := uploads[0]
	if upload.Filename != "invoice.pdf" || string(upload.Content) != "%PDF-invoice" {
		t.Errorf(
			"captured upload = %q %q, want invoice.pdf %%PDF-invoice",
			upload.Filename,
			upload.Content,
		)
	}
}

func TestUploadTaskHappyPathRoundTrip(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	client := newTestClient(t, server)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename: "receipt.pdf",
		Content:  []byte("%PDF-receipt"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	outcome, err := client.WaitForTask(context.Background(), taskID, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	if outcome.Status != paperless.TaskStatusSuccess {
		t.Errorf("status = %q, want success", outcome.Status)
	}

	if outcome.DocumentID <= 0 {
		t.Errorf("document ID = %d, want the stored document's ID", outcome.DocumentID)
	}

	checksums, listErr := client.ListDocumentChecksums(context.Background())
	if listErr != nil {
		t.Fatalf("ListDocumentChecksums: %v", listErr)
	}

	want := ChecksumOf([]byte("%PDF-receipt"))
	if _, ok := checksums[want]; !ok {
		t.Errorf("consumed document (checksum %q) missing from the document list", want)
	}
}

func TestUploadNaturalDuplicateRefusal(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	server.addDocument(Document{Title: "existing", Content: []byte("%PDF-same")})
	client := newTestClient(t, server)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename: "again.pdf",
		Content:  []byte("%PDF-same"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	outcome, err := client.WaitForTask(context.Background(), taskID, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v (duplicate refusals are honest outcomes, not errors)", err)
	}

	duplicateID, inTrash, refused := outcome.Duplicate()
	if !refused {
		t.Fatalf("outcome %+v, want a duplicate refusal", outcome)
	}

	if duplicateID != 1 || inTrash {
		t.Errorf("duplicate = (id %d, inTrash %v), want (1, false)", duplicateID, inTrash)
	}
}

func TestTaskPollUnknownTaskAnswersNotFound(t *testing.T) {
	t.Parallel()

	server := NewServer(t)

	outcome, found, err := newTestClient(t, server).GetTask(context.Background(), "task-missing")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if found {
		t.Errorf("GetTask found = true, want false for an unknown task (outcome %+v)", outcome)
	}
}

func TestUploadAppliesFormMetadata(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	client := newTestClient(t, server)

	created := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename:        "letter.pdf",
		Content:         []byte("%PDF-letter"),
		Title:           "The Letter",
		Created:         created,
		CorrespondentID: 7,
		TagIDs:          []int{2, 4},
		DocumentTypeID:  3,
		CustomFields:    []paperless.CustomFieldValue{{Field: 8, Value: "msg-99"}},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if _, err := client.WaitForTask(context.Background(), taskID, time.Millisecond); err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	upload := server.Uploads()[0]
	if upload.Title != "The Letter" || upload.Created != "2026-09-01" ||
		upload.Correspondent != "7" {
		t.Errorf(
			"captured fields = title %q created %q correspondent %q",
			upload.Title,
			upload.Created,
			upload.Correspondent,
		)
	}

	if !slices.Equal(upload.Tags, []string{"2", "4"}) || upload.DocumentType != "3" {
		t.Errorf("captured tags/type = %v/%q, want [2 4]/3", upload.Tags, upload.DocumentType)
	}

	if upload.CustomFields != `[{"field":8,"value":"msg-99"}]` {
		t.Errorf("captured custom_fields = %q", upload.CustomFields)
	}

	metas, err := client.ListDocumentMetas(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 1 {
		t.Fatalf("metas = %d entries, want the consumed document", len(metas))
	}

	meta := metas[0]
	if meta.Title != "The Letter" || meta.Correspondent != 7 || meta.DocumentTypeID != 3 {
		t.Errorf("stored meta = %+v, want title The Letter, correspondent 7, type 3", meta)
	}

	// The SDK sends created as a date-only form field, so the fake stores
	// midnight UTC of that date.
	wantCreated := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	if !meta.Created.Equal(wantCreated) {
		t.Errorf("stored created = %v, want %v", meta.Created, wantCreated)
	}

	if !slices.Equal(meta.TagIDs, []int{2, 4}) {
		t.Errorf("stored tags = %v, want [2 4]", meta.TagIDs)
	}

	if len(meta.CustomFields) != 1 || meta.CustomFields[0].Field != 8 ||
		meta.CustomFields[0].Value != "msg-99" {
		t.Errorf("stored custom fields = %v, want [{8 msg-99}]", meta.CustomFields)
	}
}

func TestUploadWithoutTitleUsesFilename(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	client := newTestClient(t, server)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename: "scan.pdf",
		Content:  []byte("%PDF-scan"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if _, err := client.WaitForTask(context.Background(), taskID, time.Millisecond); err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	metas, err := client.ListDocumentMetas(context.Background())
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 1 || metas[0].Title != "scan" {
		t.Errorf("stored title = %v, want \"scan\" (filename minus extension)", metas)
	}
}
