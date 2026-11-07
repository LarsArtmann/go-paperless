package paperless

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

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

func TestListDocumentMetasReturnsFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %q", r.URL.Query().Get("page"))
		}

		_, _ = w.Write([]byte(`{"count":2,"results":[` +
			// Legacy shape: flat top-level checksum (paperless-ngx 2.x).
			`{"id":1,"title":"Statement","correspondent":5,"created":"2026-09-03T00:00:00Z","tags":[1],"checksum":"abc"},` +
			// 3.x shape: checksum only inside versions[] (root wins), plus
			// custom fields that must round-trip into the public shape.
			`{"id":2,"title":"Other","correspondent":null,"created":"2026-09-04T00:00:00Z","tags":[],` +
			`"versions":[{"id":2,"checksum":"def","is_root":true},{"id":1,"checksum":"older","is_root":false}],` +
			`"custom_fields":[{"field":4,"value":"msg-9"}]}]}`))
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
	if first.ID.Get() != 1 || first.Title != "Statement" || first.Correspondent.Get() != 5 ||
		first.Checksum != "abc" {
		t.Errorf("first meta = %+v", first)
	}

	if len(first.TagIDs) != 1 || first.TagIDs[0].Get() != 1 {
		t.Errorf("first tags = %v", first.TagIDs)
	}

	second := metas[1]
	if second.Correspondent.Get() != 0 {
		t.Errorf("null correspondent should decode as 0, got %d", second.Correspondent)
	}

	if second.Checksum != "def" {
		t.Errorf("root version checksum = %q, want def", second.Checksum)
	}

	if len(second.CustomFields) != 1 ||
		second.CustomFields[0] != (CustomFieldValue{Field: NewCustomFieldID(4), Value: "msg-9"}) {
		t.Errorf("custom fields = %v, want [{4 msg-9}]", second.CustomFields)
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
	correspondent := NewCorrespondentID(9)
	created := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)

	err = client.UpdateDocument(t.Context(), NewDocumentID(77), UpdateDocumentRequest{
		Title:           &title,
		CorrespondentID: &correspondent,
		Created:         &created,
		TagIDs:          []TagID{NewTagID(1)},
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

// An empty (non-nil) TagIDs slice must be omitted from the PATCH body:
// Paperless-ngx replaces many-to-many fields wholesale, so a tags key with
// an empty array would wipe the document's tags as a side effect of an
// unrelated update. custom_fields must serialize field/value pairs.

// An empty (non-nil) TagIDs slice must be omitted from the PATCH body:
// Paperless-ngx replaces many-to-many fields wholesale, so a tags key with
// an empty array would wipe the document's tags as a side effect of an
// unrelated update. custom_fields must serialize field/value pairs.
func TestUpdateDocumentSendsCustomFieldsAndOmitsEmptyTags(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	title := "Retitled"

	err = client.UpdateDocument(t.Context(), NewDocumentID(42), UpdateDocumentRequest{
		Title:        &title,
		TagIDs:       []TagID{},
		CustomFields: []CustomFieldValue{{Field: NewCustomFieldID(3), Value: "msg-123"}},
	})
	if err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}

	if _, hasTags := gotBody["tags"]; hasTags {
		t.Errorf("empty TagIDs leaked into the PATCH body: %v", gotBody["tags"])
	}

	fields, ok := gotBody["custom_fields"].([]any)
	if !ok || len(fields) != 1 {
		t.Fatalf("custom_fields = %v, want one field", gotBody["custom_fields"])
	}

	field, ok := fields[0].(map[string]any)
	if !ok || field["field"] != float64(3) || field["value"] != "msg-123" {
		t.Fatalf("custom_fields[0] = %v, want field 3 with msg-123", fields[0])
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

	if err := client.UpdateDocument(
		t.Context(),
		NewDocumentID(1),
		UpdateDocumentRequest{},
	); err == nil {
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

	if err := client.DeleteDocument(t.Context(), NewDocumentID(33)); err != nil {
		t.Fatalf("DeleteDocument: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/api/documents/33/" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}
}

// The capability probe must classify all three checksum delivery shapes:
// flat (pre-3.x), versions[] (3.x), and neither (the degraded case the
// doctor warns about).

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

func TestDownloadDocumentReturnsOriginalBytes(t *testing.T) {
	t.Parallel()

	want := []byte("%PDF-1.7 original bytes")

	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write(want)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := client.DownloadDocument(t.Context(), NewDocumentID(42))
	if err != nil {
		t.Fatalf("DownloadDocument: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("body = %q, want %q", got, want)
	}

	if wantPath := "/api/documents/42/download/"; gotPath != wantPath {
		t.Fatalf("request path = %q, want %q", gotPath, wantPath)
	}
}

func TestDownloadDocumentWrapsErrorsWithDocumentID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.DownloadDocument(t.Context(), NewDocumentID(42))
	if err == nil {
		t.Fatal("expected error for missing document")
	}

	if !strings.Contains(err.Error(), "download document 42") {
		t.Fatalf("error must name the failed download, got %v", err)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family for 404, got %v (%v)", family, err)
	}
}

//nolint:dupl // the cap tests are intentionally parallel per-endpoint checks
func TestListDocumentChecksumsCapStopsAtMaxPages(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		page := r.URL.Query().Get("page")

		entries := make([]string, 0, documentListPageSize)
		for i := range documentListPageSize {
			entries = append(entries, fmt.Sprintf(
				`{"id":%d,"checksum":"sha256-page-%s-%03d"}`, i, page, i))
		}

		_, _ = w.Write([]byte(`{"results":[` + strings.Join(entries, ",") + `]}`))
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

	if len(checksums) != maxDocumentListPages*documentListPageSize {
		t.Fatalf("checksums = %d, want exactly %d (100 full pages)", len(checksums),
			maxDocumentListPages*documentListPageSize)
	}

	if requests != maxDocumentListPages {
		t.Fatalf("requests = %d, want exactly %d (the cap must stop the scan, not hang)",
			requests, maxDocumentListPages)
	}
}

//nolint:dupl // the cap tests are intentionally parallel per-endpoint checks
func TestListDocumentMetasCapStopsAtMaxPages(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		page := r.URL.Query().Get("page")

		entries := make([]string, 0, documentListPageSize)
		for i := range documentListPageSize {
			entries = append(entries, fmt.Sprintf(
				`{"id":%d,"title":"doc-%s-%03d","created":"2026-09-01"}`, i, page, i))
		}

		_, _ = w.Write([]byte(`{"results":[` + strings.Join(entries, ",") + `]}`))
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

	if len(metas) != maxDocumentListPages*documentListPageSize {
		t.Fatalf("metas = %d, want exactly %d (100 full pages)", len(metas),
			maxDocumentListPages*documentListPageSize)
	}

	if requests != maxDocumentListPages {
		t.Fatalf("requests = %d, want exactly %d (the cap must stop the scan, not hang)",
			requests, maxDocumentListPages)
	}
}

func TestListDocumentChecksumsConcurrentCalls(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[` +
			`{"id":1,"checksum":"sha256-aaa"},` +
			`{"id":2,"versions":[{"id":10,"checksum":"sha256-bbb","is_root":true}]},` +
			`{"id":3,"checksum":""}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const callers = 8

	results := make([]map[string]struct{}, callers) //nolint:makezero // pre-sized result slots
	errs := make([]error, callers)                  //nolint:makezero // pre-sized result slots

	var wg sync.WaitGroup

	for i := range callers {
		wg.Go(func() {
			results[i], errs[i] = client.ListDocumentChecksums(t.Context())
		})
	}

	wg.Wait()

	for i := range callers {
		if errs[i] != nil {
			t.Fatalf("caller %d: %v", i, errs[i])
		}

		if len(results[i]) != 2 {
			t.Fatalf(
				"caller %d got %d checksums, want 2 (empty flat checksum skipped)",
				i,
				len(results[i]),
			)
		}

		for checksum := range results[0] {
			if _, ok := results[i][checksum]; !ok {
				t.Fatalf("caller %d is missing checksum %q", i, checksum)
			}
		}
	}
}

func BenchmarkListChecksums(b *testing.B) {
	const docsPerPage = 100

	entries := make([]string, 0, docsPerPage)

	for range docsPerPage {
		entries = append(entries, `{"checksum":"bench-checksum"}`)
	}

	payload := `{"results":[` + strings.Join(entries, ",") + `],"next":null}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	ctx := b.Context()

	b.ResetTimer()

	for b.Loop() {
		if _, err := client.ListDocumentChecksums(ctx); err != nil {
			b.Fatalf("ListDocumentChecksums: %v", err)
		}
	}
}

func TestDeleteDocumentNoteSendsIDParam(t *testing.T) {
	t.Parallel()

	var gotID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}

		gotID = r.URL.Query().Get("id")

		_, _ = w.Write([]byte(`[{"id":2,"note":"remaining","created":"2026-09-13T12:00:00Z"}]`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	notes, err := client.DeleteDocumentNote(t.Context(), NewDocumentID(7), NewNoteID(3))
	if err != nil {
		t.Fatalf("DeleteDocumentNote: %v", err)
	}

	if gotID != "3" {
		t.Fatalf("delete id param = %q, want 3", gotID)
	}

	if len(notes) != 1 || notes[0].Note != "remaining" {
		t.Fatalf("notes = %+v, want the server's updated list", notes)
	}
}

func TestDeleteDocumentSurfacesServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = client.DeleteDocument(t.Context(), NewDocumentID(12))
	if err == nil {
		t.Fatal("expected the server error to surface")
	}

	if !strings.Contains(err.Error(), "12") {
		t.Errorf("error should name the document ID: %v", err)
	}
}

// The self-heal PATCH failure must surface with the tag name — the one
// updateMatchingAlgorithm branch no other test drives.
