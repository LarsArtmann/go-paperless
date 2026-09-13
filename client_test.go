package paperless

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestNewRejectsMissingConfig(t *testing.T) {
	t.Parallel()

	if _, err := New("", "token"); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("empty URL: expected ErrInvalidConfig, got %v", err)
	}

	if _, err := New("https://paperless.example.com", ""); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("empty token: expected ErrInvalidConfig, got %v", err)
	}
}

func TestNewRejectsNonHTTPScheme(t *testing.T) {
	t.Parallel()

	_, err := New("ftp://paperless.example.com", "token")
	if err == nil {
		t.Fatal("expected error for ftp scheme")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v", errorfamily.Classify(err))
	}
}

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
		TagIDs:   []int{3, 7},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if taskID != "4f7e8a2b-1234-4cde-9abc-def012345678" {
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

func TestEnsureTagFindsExisting(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags/" || r.Method != http.MethodGet {
			t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}

		if name := r.URL.Query().Get("name__iexact"); name != "inboxclean" {
			t.Errorf("name__iexact = %q", name)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":42,"name":"inboxclean"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureTag(t.Context(), "inboxclean")
	if err != nil {
		t.Fatalf("EnsureTag: %v", err)
	}

	if id != 42 {
		t.Fatalf("id = %d, want 42", id)
	}
}

func TestEnsureTagSelfHealsLegacyAutoTag(t *testing.T) {
	t.Parallel()

	patched := make(chan *namedPayload, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags/":
			// A tag created by an older InboxClean release: matching_algorithm
			// "auto" (6), the exact shape the 2026-09 classifier incident left
			// behind on live installs.
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":42,"name":"gmail","matching_algorithm":6}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/tags/42/":
			var payload namedPayload
			if err := json.UnmarshalRead(r.Body, &payload); err != nil {
				t.Errorf("decode patch body: %v", err)
			}

			patched <- &payload

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":42,"name":"gmail","matching_algorithm":0}`))
		default:
			t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureTag(t.Context(), "gmail")
	if err != nil {
		t.Fatalf("EnsureTag: %v", err)
	}

	if id != 42 {
		t.Fatalf("id = %d, want 42", id)
	}

	select {
	case payload := <-patched:
		if payload.MatchingAlgorithm != matchingAlgorithmNone {
			t.Fatalf("patched matching_algorithm = %d, want none (0)", payload.MatchingAlgorithm)
		}
	default:
		t.Fatal("legacy auto tag was not demoted via PATCH")
	}
}

func TestEnsureTagLeavesNonAutoTagAlone(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Any method other than the exact-name GET means EnsureTag tried to
		// mutate a tag it does not own (user-configured matching algorithms
		// must be preserved).
		if r.URL.Path != "/api/tags/" || r.Method != http.MethodGet {
			t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":9,"name":"gmail","matching_algorithm":1}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureTag(t.Context(), "gmail")
	if err != nil {
		t.Fatalf("EnsureTag: %v", err)
	}

	if id != 9 {
		t.Fatalf("id = %d, want 9", id)
	}
}

func TestEnsureTagCreatesWhenMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/tags/":
			var payload namedPayload
			if err := json.UnmarshalRead(r.Body, &payload); err != nil {
				t.Errorf("decode create body: %v", err)
			}

			if payload.Name != "newtag" {
				t.Errorf("created name = %q", payload.Name)
			}

			// Tags are provenance metadata: they must never join the
			// classifier's training set (auto matching would leak the tag onto
			// similar-looking manual scans).
			if payload.MatchingAlgorithm != matchingAlgorithmNone {
				t.Errorf("matching_algorithm = %d, want none (0)", payload.MatchingAlgorithm)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":7,"name":"newtag","matching_algorithm":0}`))
		default:
			t.Errorf("unexpected call: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureTag(t.Context(), "newtag")
	if err != nil {
		t.Fatalf("EnsureTag: %v", err)
	}

	if id != 7 {
		t.Fatalf("id = %d, want 7", id)
	}
}

func TestPingRejectsBadToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pathDocuments {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		if r.Header.Get("Authorization") != "Token bad" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client, err := New(server.URL, "bad")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	pingErr := client.Ping(t.Context())
	if pingErr == nil {
		t.Fatal("expected ping error")
	}

	if family := errorfamily.Classify(pingErr); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, pingErr)
	}
}

func TestPingSucceeds(t *testing.T) {
	t.Parallel()

	var gotPath, gotQuery, gotAccept string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAccept = r.Header.Get("Accept")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[],"count":0}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.Ping(t.Context()); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	if gotPath != pathDocuments {
		t.Fatalf("ping path = %q, want %q", gotPath, pathDocuments)
	}

	if !strings.Contains(gotQuery, "page=1") || !strings.Contains(gotQuery, "page_size=1") {
		t.Fatalf("ping query = %q, want page=1 and page_size=1", gotQuery)
	}

	if gotAccept != "application/json; version="+apiVersion {
		t.Fatalf("ping Accept = %q, want versioned JSON", gotAccept)
	}
}

// TestPingAgainstHtmlOnlyApiRoot pins the live Paperless-ngx contract that
// broke production (paperless 3.0.5): the API root serves browsable HTML
// only and answers any JSON Accept header with 406 regardless of the token,
// while the document list view is the authenticated JSON endpoint. Ping
// must survive exactly this server shape.
func TestPingAgainstHtmlOnlyApiRoot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/":
			w.WriteHeader(http.StatusNotAcceptable)
		case pathDocuments:
			if r.Header.Get("Authorization") != "Token good" {
				w.WriteHeader(http.StatusUnauthorized)

				return
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results":[{"checksum":"abc"}],"count":1}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "good")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.Ping(t.Context()); err != nil {
		t.Fatalf("Ping against HTML-only API root: %v", err)
	}
}

func TestListDocumentChecksumsPaginates(t *testing.T) {
	t.Parallel()

	// A full first page (documentListPageSize entries) forces a second
	// request; the short second page terminates the scan. Page one serves
	// the legacy flat checksum, page two the paperless-ngx 3.x shape where
	// the checksum lives in versions[] only.
	pageOne := buildDocumentPage(documentListPageSize)
	pageTwo := `{"results":[{"id":99,"versions":[{"id":99,"checksum":"sha-final","is_root":true}]}]}`

	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/documents/" {
			t.Errorf("path = %q, want /api/documents/", r.URL.Path)
		}

		requests++

		if r.URL.Query().Get("page") == "1" {
			_, _ = w.Write([]byte(pageOne))

			return
		}

		_, _ = w.Write([]byte(pageTwo))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	checksums, err := client.ListDocumentChecksums(t.Context())
	if err != nil {
		t.Fatalf("ListDocumentChecksums: %v", err)
	}

	if len(checksums) != documentListPageSize+1 {
		t.Fatalf("checksums = %d entries, want %d", len(checksums), documentListPageSize+1)
	}

	if _, ok := checksums["sha-final"]; !ok {
		t.Error("checksum set missing second-page entry")
	}

	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (short page ends the scan)", requests)
	}
}

// buildDocumentPage renders one full page of document entries with distinct
// checksums.
func buildDocumentPage(count int) string {
	entries := make([]string, count) //nolint:makezero // pre-sized

	for i := range count {
		entries[i] = fmt.Sprintf(`{"checksum":"sha-%d"}`, i)
	}

	return `{"results":[` + strings.Join(entries, ",") + `]}`
}

func TestListDocumentChecksumsSinglePage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A short first page ends the scan immediately.
		_, _ = w.Write([]byte(`{"results":[{"checksum":"sha-only"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	checksums, err := client.ListDocumentChecksums(t.Context())
	if err != nil {
		t.Fatalf("ListDocumentChecksums: %v", err)
	}

	if len(checksums) != 1 {
		t.Fatalf("checksums = %v, want 1 entry", checksums)
	}
}

func TestListDocumentChecksumsRejectsBadToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := New(server.URL, "bad")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, listErr := client.ListDocumentChecksums(t.Context())
	if listErr == nil {
		t.Fatal("expected error for unauthorized token")
	}

	if family := errorfamily.Classify(listErr); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, listErr)
	}
}

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

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{"seconds", "120", 2 * time.Minute, true},
		{"zero seconds", "0", 0, true},
		{"negative seconds invalid", "-5", 0, false},
		{"empty invalid", "", 0, false},
		{"garbage invalid", "soon", 0, false},
		{"future date", now.Add(time.Minute).Format(http.TimeFormat), time.Minute, true},
		{"past date retries now", now.Add(-time.Hour).Format(http.TimeFormat), 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := parseRetryAfter(tc.value, now)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v (got %s)", ok, tc.ok, got)
			}

			if ok && got != tc.want {
				t.Fatalf("delay = %s, want %s", got, tc.want)
			}
		})
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
		CorrespondentID: 42,
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

func TestEnsureCorrespondentFindsExisting(t *testing.T) {
	t.Parallel()

	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/correspondents/" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}

		gotQuery = r.URL.Query().Get("name__iexact")

		_, _ = w.Write(
			[]byte(`{"count":1,"results":[{"id":7,"name":"ACME","matching_algorithm":6}]}`),
		)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureCorrespondent(t.Context(), "acme")
	if err != nil {
		t.Fatalf("EnsureCorrespondent: %v", err)
	}

	if gotQuery != "acme" {
		t.Errorf("name__iexact = %q, want %q", gotQuery, "acme")
	}

	if id != 7 {
		t.Errorf("id = %d, want 7", id)
	}
}

func TestEnsureCorrespondentCreatesWhenMissing(t *testing.T) {
	t.Parallel()

	created := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"count":0,"results":[]}`))
		case http.MethodPost:
			created = true

			var payload struct {
				Name              string `json:"name"`
				MatchingAlgorithm int    `json:"matching_algorithm"` //nolint:tagliatelle // Paperless-ngx serves snake_case
			}

			if err := json.UnmarshalRead(r.Body, &payload); err != nil {
				t.Errorf("decode payload: %v", err)
			}

			if payload.Name != "Alior Bank" {
				t.Errorf("created name = %q", payload.Name)
			}

			if payload.MatchingAlgorithm != matchingAlgorithmAuto {
				t.Errorf(
					"matching_algorithm = %d, want auto (%d)",
					payload.MatchingAlgorithm,
					matchingAlgorithmAuto,
				)
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":12,"name":"Alior Bank","matching_algorithm":6}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureCorrespondent(t.Context(), "Alior Bank")
	if err != nil {
		t.Fatalf("EnsureCorrespondent: %v", err)
	}

	if !created {
		t.Error("correspondent was not created")
	}

	if id != 12 {
		t.Errorf("id = %d, want 12", id)
	}
}

func TestListDocumentMetasReturnsFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %q", r.URL.Query().Get("page"))
		}

		_, _ = w.Write([]byte(`{"count":2,"results":[` +
			// Legacy shape: flat top-level checksum (paperless-ngx 2.x).
			`{"id":1,"title":"Statement","correspondent":5,"created":"2026-09-03T00:00:00Z","tags":[1],"checksum":"abc"},` +
			// 3.x shape: checksum only inside versions[] (root wins).
			`{"id":2,"title":"Other","correspondent":null,"created":"2026-09-04T00:00:00Z","tags":[],"versions":[{"id":2,"checksum":"def","is_root":true},{"id":1,"checksum":"older","is_root":false}]}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	metas, err := client.ListDocumentMetas(t.Context())
	if err != nil {
		t.Fatalf("ListDocumentMetas: %v", err)
	}

	if len(metas) != 2 {
		t.Fatalf("metas = %d entries, want 2", len(metas))
	}

	first := metas[0]
	if first.ID != 1 || first.Title != "Statement" || first.Correspondent != 5 ||
		first.Checksum != "abc" {
		t.Errorf("first meta = %+v", first)
	}

	if len(first.TagIDs) != 1 || first.TagIDs[0] != 1 {
		t.Errorf("first tags = %v", first.TagIDs)
	}

	second := metas[1]
	if second.Correspondent != 0 {
		t.Errorf("null correspondent should decode as 0, got %d", second.Correspondent)
	}
}

func TestUpdateDocumentSendsPatchBody(t *testing.T) {
	t.Parallel()

	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	title := "Wyciąg z rachunku"
	correspondent := 9
	created := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)

	err = client.UpdateDocument(t.Context(), 77, UpdateDocumentRequest{
		Title:           &title,
		CorrespondentID: &correspondent,
		Created:         &created,
		TagIDs:          []int{1},
	})
	if err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}

	if gotPath != "/api/documents/77/" {
		t.Errorf("path = %q", gotPath)
	}

	if gotBody["title"] != title {
		t.Errorf("title = %v", gotBody["title"])
	}

	if gotBody["correspondent"] != float64(9) {
		t.Errorf("correspondent = %v", gotBody["correspondent"])
	}

	if gotBody["created"] != "2026-09-03" {
		t.Errorf("created = %v, want date-only", gotBody["created"])
	}

	tags, ok := gotBody["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != float64(1) {
		t.Errorf("tags = %v", gotBody["tags"])
	}
}

func TestUpdateDocumentRejectsEmptyRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("empty update must not reach the server")
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.UpdateDocument(t.Context(), 1, UpdateDocumentRequest{}); err == nil {
		t.Fatal("expected rejection for empty update request")
	}
}

func TestDeleteDocumentSendsDelete(t *testing.T) {
	t.Parallel()

	var (
		gotMethod string
		gotPath   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.DeleteDocument(t.Context(), 33); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/api/documents/33/" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}
}

// The capability probe must classify all three checksum delivery shapes:
// flat (pre-3.x), versions[] (3.x), and neither (the degraded case the
// doctor warns about).
func TestProbeCapabilitiesDetectsChecksumShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantShape  string
		wantFlat   bool
		wantVers   bool
		wantSample int
	}{
		{
			name:       "flat pre-3x",
			body:       `{"results":[{"id":1,"checksum":"sha-flat"}]}`,
			wantShape:  "flat (pre-3.x)",
			wantFlat:   true,
			wantSample: 1,
		},
		{
			name:       "versions 3x",
			body:       `{"results":[{"id":2,"versions":[{"id":2,"checksum":"sha-vers","is_root":true}]}]}`,
			wantShape:  "versions[] (3.x)",
			wantVers:   true,
			wantSample: 1,
		},
		{
			name:       "both shapes",
			body:       `{"results":[{"id":3,"checksum":"a","versions":[{"id":3,"checksum":"b","is_root":true}]},{"id":4,"versions":[{"id":4,"checksum":"c","is_root":true}]}]}`,
			wantShape:  "flat+versions[]",
			wantFlat:   true,
			wantVers:   true,
			wantSample: 2,
		},
		{
			name:      "neither shape",
			body:      `{"results":[{"id":5,"title":"no checksums here"}]}`,
			wantShape: "none",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != pathDocuments {
						w.WriteHeader(http.StatusNotFound)

						return
					}

					w.Header().Set("Content-Type", "application/json; version=10")
					_, _ = w.Write([]byte(tc.body))
				}),
			)
			defer server.Close()

			client, err := New(server.URL, "token")
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			caps, err := client.ProbeCapabilities(t.Context())
			if err != nil {
				t.Fatalf("ProbeCapabilities: %v", err)
			}

			if caps.ChecksumShape() != tc.wantShape {
				t.Errorf("shape = %q, want %q", caps.ChecksumShape(), tc.wantShape)
			}

			if caps.FlatChecksum != tc.wantFlat || caps.VersionedChecksum != tc.wantVers {
				t.Errorf("flat=%v versions=%v, want flat=%v versions=%v",
					caps.FlatChecksum, caps.VersionedChecksum, tc.wantFlat, tc.wantVers)
			}

			if caps.DocumentsSampled != tc.wantSample {
				t.Errorf("sampled = %d, want %d", caps.DocumentsSampled, tc.wantSample)
			}

			if caps.AcceptAPIVersion != "10" {
				t.Errorf("AcceptAPIVersion = %q, want 10", caps.AcceptAPIVersion)
			}
		})
	}
}

func TestParseDocumentCreated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want time.Time
	}{
		{
			name: "rfc3339 with UTC offset",
			raw:  "2026-09-06T14:30:05Z",
			want: time.Date(2026, 9, 6, 14, 30, 5, 0, time.UTC),
		},
		{
			name: "rfc3339 with numeric offset",
			raw:  "2026-09-06T16:30:05+02:00",
			want: time.Date(2026, 9, 6, 16, 30, 5, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name: "timezone-less datetime",
			raw:  "2026-09-06T14:30:05",
			want: time.Date(2026, 9, 6, 14, 30, 5, 0, time.UTC),
		},
		{
			name: "bare date (upload shape)",
			raw:  "2026-09-06",
			want: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "empty string",
			raw:  "",
			want: time.Time{},
		},
		{
			name: "garbage",
			raw:  "not-a-date",
			want: time.Time{},
		},
		{
			name: "date with time but no seconds",
			raw:  "2026-09-06T14:30",
			want: time.Time{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := parseDocumentCreated(tc.raw)

			if !got.Equal(tc.want) {
				t.Fatalf("parseDocumentCreated(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestGetCorrespondentName(t *testing.T) {
	t.Parallel()

	var requestedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path

		if r.URL.Path == "/api/correspondents/7/" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":7,"name":"Alior Bank"}`))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	name, err := client.GetCorrespondentName(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetCorrespondentName(7): %v", err)
	}

	if name != "Alior Bank" {
		t.Fatalf("name = %q, want %q", name, "Alior Bank")
	}

	if requestedPath != "/api/correspondents/7/" {
		t.Fatalf("request path = %q, want the detail endpoint", requestedPath)
	}

	if _, err := client.GetCorrespondentName(context.Background(), 99); err == nil {
		t.Fatal("expected an error for a missing correspondent")
	}
}

func newEnsureTestClient(
	t *testing.T,
	routes map[string]func(w http.ResponseWriter, r *http.Request),
) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := routes[r.URL.Path]; ok {
			handler(w, r)

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return client
}

func TestEnsureDocumentTypeFindsAndCreates(t *testing.T) {
	t.Parallel()

	documentTypes := map[string]int{"Email": 3}

	client := newEnsureTestClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/api/document_types/": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method %s before creation", r.Method)
			}

			name := r.URL.Query().Get("name__iexact")
			if id, ok := documentTypes[name]; ok {
				_, _ = fmt.Fprintf(
					w,
					`{"results":[{"id":%d,"name":%q,"matching_algorithm":0}]}`,
					id,
					name,
				)

				return
			}

			_, _ = w.Write([]byte(`{"results":[]}`))
		},
		"/api/document_types/create": func(w http.ResponseWriter, r *http.Request) {
			t.Error("EnsureDocumentType must find the existing type, not create")
		},
	})

	id, err := client.EnsureDocumentType(context.Background(), "Email")
	if err != nil {
		t.Fatalf("EnsureDocumentType: %v", err)
	}

	if id != 3 {
		t.Fatalf("id = %d, want 3", id)
	}
}

func TestGetDocumentTypeName(t *testing.T) {
	t.Parallel()

	client := newEnsureTestClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/api/document_types/5/": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":5,"name":"Invoice"}`))
		},
		"/api/document_types/9/": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	})

	name, err := client.GetDocumentTypeName(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetDocumentTypeName(5): %v", err)
	}

	if name != "Invoice" {
		t.Fatalf("name = %q, want Invoice", name)
	}

	if _, err := client.GetDocumentTypeName(context.Background(), 9); err == nil {
		t.Fatal("expected an error for a missing document type")
	}
}

func TestEnsureAndFindCustomField(t *testing.T) {
	t.Parallel()

	fields := map[string]int{}

	client := newEnsureTestClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/api/custom_fields/": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				name := r.URL.Query().Get("name__iexact")
				if id, ok := fields[name]; ok {
					_, _ = fmt.Fprintf(
						w,
						`{"results":[{"id":%d,"name":%q,"data_type":"string"}]}`,
						id,
						name,
					)

					return
				}

				_, _ = w.Write([]byte(`{"results":[]}`))
			case http.MethodPost:
				var payload struct {
					Name string `json:"name"`
					//nolint:tagliatelle // Paperless-ngx serves snake_case JSON keys
					DataType string `json:"data_type"`
				}

				_ = json.UnmarshalRead(r.Body, &payload)

				if payload.DataType != "string" {
					t.Errorf("created data_type = %q, want string", payload.DataType)
				}

				id := len(fields) + 1
				fields[payload.Name] = id

				w.WriteHeader(http.StatusCreated)
				_, _ = fmt.Fprintf(
					w,
					`{"id":%d,"name":%q,"data_type":%q}`,
					id,
					payload.Name,
					payload.DataType,
				)
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		},
	})

	id, err := client.EnsureCustomField(context.Background(), "gmail_message_id")
	if err != nil {
		t.Fatalf("EnsureCustomField: %v", err)
	}

	if id != 1 {
		t.Fatalf("first ensure id = %d, want 1", id)
	}

	id, err = client.EnsureCustomField(context.Background(), "gmail_message_id")
	if err != nil {
		t.Fatalf("second EnsureCustomField: %v", err)
	}

	if id != 1 {
		t.Fatalf("second ensure id = %d, want the same field 1", id)
	}

	found, exists, err := client.FindCustomField(context.Background(), "gmail_message_id")
	if err != nil || !exists || found != 1 {
		t.Fatalf("FindCustomField = (%d, %v, %v), want (1, true, nil)", found, exists, err)
	}

	_, exists, err = client.FindCustomField(context.Background(), "missing_field")
	if err != nil || exists {
		t.Fatalf("FindCustomField missing = (%d, %v, %v), want (0, false, nil)", 0, exists, err)
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

	_, err = client.Upload(context.Background(), UploadRequest{
		Filename:       "invoice.pdf",
		Content:        []byte("%PDF-1.4"),
		DocumentTypeID: 3,
		CustomFields:   []CustomFieldValue{{Field: 1, Value: "msg-123"}},
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

func TestGetTaskClassifiesConsumedDocument(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotPath := r.URL.Path; gotPath != "/api/tasks/" {
			t.Errorf("path = %q, want /api/tasks/", gotPath)
		}

		if gotQuery := r.URL.Query().Get("task_id"); gotQuery != "task-uuid-1" {
			t.Errorf("task_id = %q, want task-uuid-1", gotQuery)
		}

		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[` +
			`{"task_id":"task-uuid-1","status":"success","result_data":{"document_id":42},` +
			`"related_document_ids":[42]}]}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, found, err := client.GetTask(t.Context(), "task-uuid-1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if !found {
		t.Fatal("expected the task to be found")
	}

	if outcome.Status != TaskStatusSuccess || !outcome.Status.Terminal() {
		t.Fatalf("status = %q, want terminal success", outcome.Status)
	}

	if outcome.DocumentID != 42 {
		t.Fatalf("DocumentID = %d, want 42", outcome.DocumentID)
	}

	if outcome.DuplicateRefused {
		t.Fatal("consumed task must not be classified as a duplicate refusal")
	}
}

func TestGetTaskClassifiesDuplicateRefusal(t *testing.T) {
	t.Parallel()

	// paperless-ngx 3.x: the consumer returns ConsumeFileDuplicateResult and
	// the postrun handler demotes the task to failure with result_data
	// {"duplicate_of": N, "duplicate_in_trash": bool}.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[` +
			`{"task_id":"task-uuid-2","status":"failure",` +
			`"result_data":{"duplicate_of":7,"duplicate_in_trash":false},` +
			`"related_document_ids":[7]}]}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, found, err := client.GetTask(t.Context(), "task-uuid-2")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if !found {
		t.Fatal("expected the task to be found")
	}

	if outcome.Status != TaskStatusFailure || !outcome.Status.Terminal() {
		t.Fatalf("status = %q, want terminal failure", outcome.Status)
	}

	if !outcome.DuplicateRefused {
		t.Fatal("expected DuplicateRefused")
	}

	if outcome.DocumentID != 7 {
		t.Fatalf("DocumentID = %d, want the duplicate's 7", outcome.DocumentID)
	}
}

func TestGetTaskClassifiesGenericFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[` +
			`{"task_id":"task-uuid-3","status":"failure",` +
			`"result_data":{"error_type":"ParseError","error_message":"bad pdf"}}]}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, found, err := client.GetTask(t.Context(), "task-uuid-3")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if !found {
		t.Fatal("expected the task to be found")
	}

	if !outcome.Status.Terminal() {
		t.Fatalf("status %q should be terminal", outcome.Status)
	}

	if outcome.DuplicateRefused {
		t.Fatal("a parse failure is not a duplicate refusal")
	}

	if outcome.ErrorMessage != "bad pdf" {
		t.Fatalf("ErrorMessage = %q, want bad pdf", outcome.ErrorMessage)
	}

	if outcome.DocumentID != 0 {
		t.Fatalf("DocumentID = %d, want 0", outcome.DocumentID)
	}
}

func TestGetTaskNonTerminalPending(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":1,"next":null,"previous":null,"results":[` +
			`{"task_id":"task-uuid-4","status":"started","result_data":null}]}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, found, err := client.GetTask(t.Context(), "task-uuid-4")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if !found {
		t.Fatal("expected the task to be found")
	}

	if outcome.Status.Terminal() {
		t.Fatalf("status %q must not be terminal", outcome.Status)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"count":0,"next":null,"previous":null,"results":[]}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, found, err := client.GetTask(t.Context(), "missing-uuid")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if found {
		t.Fatal("expected found=false for an empty result set")
	}
}

func TestGetTaskToleratesBareArray(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"task_id":"task-uuid-5","status":"SUCCESS",` +
			`"result_data":{"document_id":9}}]`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, found, err := client.GetTask(t.Context(), "task-uuid-5")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if !found {
		t.Fatal("expected the task to be found")
	}

	if outcome.Status != TaskStatusSuccess || outcome.DocumentID != 9 {
		t.Fatalf("outcome = %+v, want success with document 9", outcome)
	}
}

func TestGetTaskRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, _, err = client.GetTask(t.Context(), "task-uuid-6")
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}
