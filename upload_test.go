package paperless

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestUploadSendsMultipartWithDocumentField(t *testing.T) {
	t.Parallel()

	var (
		gotAuth        string
		gotAccept      string
		gotMethod      string
		gotPath        string
		gotFormFile    bool
		gotTitle       string
		gotCreated     string
		gotTagFields   []string
		gotFileContent []byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotMethod = r.Method
		gotPath = r.URL.Path

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		file, header, err := r.FormFile("document")
		if err == nil {
			gotFormFile = true
			gotFileContent, _ = io.ReadAll(file)
			_ = file.Close()
			_ = header
		}

		gotTitle = r.FormValue("title")
		gotCreated = r.FormValue("created")
		gotTagFields = r.Form["tags"]

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`"4f7e8a2b-1234-4cde-9abc-def012345678"`))
	}))
	defer server.Close()

	client, err := New(server.URL, "secret-token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	taskID, err := client.Upload(t.Context(), UploadRequest{
		Filename: "invoice.pdf",
		Content:  []byte("%PDF-1.4 fake"),
		Title:    "Invoice from ACME",
		Created:  time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC),
		TagIDs:   []TagID{NewTagID(3), NewTagID(7)},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if taskID.Get() != "4f7e8a2b-1234-4cde-9abc-def012345678" {
		t.Fatalf("task ID not unquoted/trimmed: %q", taskID)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}

	if gotPath != "/api/documents/post_document/" {
		t.Errorf("path = %q", gotPath)
	}

	if gotAuth != "Token secret-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}

	if gotAccept != "application/json; version=10" {
		t.Errorf("Accept = %q", gotAccept)
	}

	if !gotFormFile {
		t.Error("multipart field 'document' missing")
	}

	if string(gotFileContent) != "%PDF-1.4 fake" {
		t.Errorf("file content = %q", gotFileContent)
	}

	if gotTitle != "Invoice from ACME" {
		t.Errorf("title = %q", gotTitle)
	}

	if gotCreated != "2026-08-18" {
		t.Errorf("created = %q", gotCreated)
	}

	if len(gotTagFields) != 2 || gotTagFields[0] != "3" || gotTagFields[1] != "7" {
		t.Errorf("tags = %v", gotTagFields)
	}
}

func TestUploadClassifiesAuthFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := New(server.URL, "wrong-token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{Filename: "x.pdf", Content: []byte("x")})
	if err == nil {
		t.Fatal("expected error")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", errorfamily.Classify(err), err)
	}
}

func TestUploadClassifiesServerErrorAsTransient(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{Filename: "x.pdf", Content: []byte("x")})
	if err == nil {
		t.Fatal("expected error")
	}

	if errorfamily.Classify(err) != errorfamily.Transient {
		t.Fatalf("expected Transient family, got %v (%v)", errorfamily.Classify(err), err)
	}
}

func TestUploadMissingTaskIDClassifiedAsCorruption(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`""`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{Filename: "x.pdf", Content: []byte("x")})
	if err == nil {
		t.Fatal("expected error when the server returns no task ID")
	}

	coded, ok := errors.AsType[*errorfamily.Error](err)
	if !ok {
		t.Fatalf("expected *errorfamily.Error, got %T (%v)", err, err)
	}

	if coded.Code() != "paperless.missing_task_id" {
		t.Fatalf("code = %q, want paperless.missing_task_id", coded.Code())
	}

	if coded.Family() != errorfamily.Corruption {
		t.Fatalf(
			"family = %v, want Corruption (the server misbehaved, not the caller)",
			coded.Family(),
		)
	}
}

//nolint:dupl // the 429 and 503 tests are intentionally parallel wire-shape checks
func TestUpload429CarriesRetryAfterHint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{Filename: "x.pdf", Content: []byte("x")})
	if err == nil {
		t.Fatal("expected error")
	}

	hint, ok := errors.AsType[*RetryAfterError](err)
	if !ok {
		t.Fatalf("expected RetryAfterError, got %T (%v)", err, err)
	}

	if hint.After != 7*time.Second {
		t.Fatalf("hint = %s, want 7s", hint.After)
	}

	if errorfamily.Classify(err) != errorfamily.Transient {
		t.Fatalf("wrapped family must stay Transient, got %v", errorfamily.Classify(err))
	}
}

//nolint:dupl // the 429 and 503 tests are intentionally parallel wire-shape checks
func TestUpload503CarriesRetryAfterHint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{Filename: "x.pdf", Content: []byte("x")})
	if err == nil {
		t.Fatal("expected error")
	}

	hint, ok := errors.AsType[*RetryAfterError](err)
	if !ok {
		t.Fatalf("expected RetryAfterError, got %T (%v)", err, err)
	}

	if hint.After != 12*time.Second {
		t.Fatalf("hint = %s, want 12s", hint.After)
	}

	if errorfamily.Classify(err) != errorfamily.Transient {
		t.Fatalf("wrapped family must stay Transient, got %v", errorfamily.Classify(err))
	}
}

func TestUploadSendsCorrespondentField(t *testing.T) {
	t.Parallel()

	var gotCorrespondent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		gotCorrespondent = r.FormValue("correspondent")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`"task-1"`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.Upload(t.Context(), UploadRequest{
		Filename:        "invoice.pdf",
		Content:         []byte("%PDF"),
		CorrespondentID: NewCorrespondentID(42),
	}); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if gotCorrespondent != "42" {
		t.Errorf("correspondent form field = %q, want %q", gotCorrespondent, "42")
	}
}

func TestUploadOmitsCorrespondentWhenUnset(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if _, ok := r.MultipartForm.Value["correspondent"]; ok {
			t.Error("correspondent field sent without CorrespondentID")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`"task-1"`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.Upload(t.Context(), UploadRequest{
		Filename: "invoice.pdf",
		Content:  []byte("%PDF"),
	}); err != nil {
		t.Fatalf("Upload: %v", err)
	}
}

func TestUploadSendsDocumentTypeAndCustomFields(t *testing.T) {
	t.Parallel()

	var (
		gotDocumentType string
		gotCustomFields string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		gotDocumentType = r.FormValue("document_type")
		gotCustomFields = r.FormValue("custom_fields")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`"task-1"`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{
		Filename:       "invoice.pdf",
		Content:        []byte("%PDF-1.4"),
		DocumentTypeID: NewDocumentTypeID(3),
		CustomFields:   []CustomFieldValue{{Field: NewCustomFieldID(1), Value: "msg-123"}},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if gotDocumentType != "3" {
		t.Fatalf("document_type = %q, want 3", gotDocumentType)
	}

	if gotCustomFields != `[{"field":1,"value":"msg-123"}]` {
		t.Fatalf("custom_fields = %q, want the JSON array", gotCustomFields)
	}
}

func BenchmarkUpload(b *testing.B) {
	body := bytes.Repeat([]byte("%PDF-1.4 payload "), 4096) // ~64 KiB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"task_id":"bench-task","status":"pending"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	ctx := b.Context()

	b.ResetTimer()

	for b.Loop() {
		if _, err := client.Upload(ctx, UploadRequest{
			Filename: "bench.pdf",
			Content:  body,
			Title:    "Benchmark",
		}); err != nil {
			b.Fatalf("Upload: %v", err)
		}
	}
}

// errPartWriteFailed is the sentinel failingPartWriter always returns.
var errPartWriteFailed = errors.New("part write failed")

// failingPartWriter makes every multipart part write fail, forcing the
// error branches in the upload-metadata writers without a live server.

func TestWriteUploadMetadataFieldFailuresCarryCodedErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		req      UploadRequest
		wantCode string
	}{
		{
			name:     "title",
			req:      UploadRequest{Title: "Invoice"},
			wantCode: "paperless.write_title",
		},
		{
			name:     "created",
			req:      UploadRequest{Created: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)},
			wantCode: "paperless.write_created",
		},
		{
			name:     "correspondent",
			req:      UploadRequest{CorrespondentID: NewCorrespondentID(7)},
			wantCode: "paperless.write_correspondent",
		},
		{
			name:     "tags",
			req:      UploadRequest{TagIDs: []TagID{NewTagID(3)}},
			wantCode: "paperless.write_tags",
		},
		{
			name:     "document_type",
			req:      UploadRequest{DocumentTypeID: NewDocumentTypeID(9)},
			wantCode: "paperless.write_document_type",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			writer := multipart.NewWriter(failingPartWriter{})

			err := writeUploadMetadata(writer, testCase.req)
			if err == nil {
				t.Fatal("expected a write failure against the closed writer")
			}

			coded, ok := errors.AsType[*errorfamily.Error](err)
			if !ok {
				t.Fatalf("expected *errorfamily.Error, got %T (%v)", err, err)
			}

			if coded.Code() != testCase.wantCode {
				t.Fatalf("code = %q, want %q", coded.Code(), testCase.wantCode)
			}

			if coded.Family() != errorfamily.Infrastructure {
				t.Fatalf("family = %v, want Infrastructure", coded.Family())
			}
		})
	}
}

func TestWriteCustomFieldsFieldWriteFailureCarriesCodedError(t *testing.T) {
	t.Parallel()

	writer := multipart.NewWriter(failingPartWriter{})

	err := writeCustomFieldsField(
		writer,
		[]CustomFieldValue{{Field: NewCustomFieldID(1), Value: "gmail-msg-42"}},
	)
	if err == nil {
		t.Fatal("expected a write failure against the closed writer")
	}

	coded, ok := errors.AsType[*errorfamily.Error](err)
	if !ok {
		t.Fatalf("expected *errorfamily.Error, got %T (%v)", err, err)
	}

	if coded.Code() != "paperless.write_custom_fields" {
		t.Fatalf("code = %q, want paperless.write_custom_fields", coded.Code())
	}

	if coded.Family() != errorfamily.Infrastructure {
		t.Fatalf("family = %v, want Infrastructure", coded.Family())
	}
}

// TestUploadOmitsOptionalFieldsWhenZero pins the inverse multipart
// branches: a zero Created omits the created field entirely, and a
// correspondent/document-type/custom-fields-free request carries none of
// their parts (the consumption endpoint treats absent as unset, so
// sending empty strings would clobber server-side defaults).
func TestUploadOmitsOptionalFieldsWhenZero(t *testing.T) {
	t.Parallel()

	var form url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)

		form = r.Form

		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(`"task-omit"`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{
		Filename: "minimal.pdf",
		Content:  []byte("%PDF-1.4"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	for _, absent := range []string{"created", "correspondent", "document_type", "custom_fields", "title"} {
		if form.Has(absent) {
			t.Errorf("form carries %q — zero-value optional fields must be omitted, not sent empty", absent)
		}
	}
}
