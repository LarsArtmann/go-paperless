package paperless

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

//nolint:dupl // the cap tests are intentionally parallel per-endpoint checks
func TestListShareLinksCapStopsAtMaxPages(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		page := r.URL.Query().Get("page")

		entries := make([]string, 0, documentListPageSize)
		for i := range documentListPageSize {
			entries = append(entries, fmt.Sprintf(
				`{"id":%d,"slug":"page-%s-%03d","document":7}`, i, page, i))
		}

		_, _ = w.Write([]byte(`{"results":[` + strings.Join(entries, ",") + `]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	links, err := client.ListShareLinks(t.Context())
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}

	if len(links) != maxDocumentListPages*documentListPageSize {
		t.Fatalf("links = %d, want exactly %d (100 full pages)", len(links),
			maxDocumentListPages*documentListPageSize)
	}

	if requests != maxDocumentListPages {
		t.Fatalf("requests = %d, want exactly %d (the cap must stop the scan, not hang)",
			requests, maxDocumentListPages)
	}
}

func TestListShareLinksPaginates(t *testing.T) {
	t.Parallel()

	var requestedPages []string

	pageEntries := func(start, end, document int, version string) string {
		entries := make([]string, 0, end-start)
		for id := start; id < end; id++ {
			entries = append(entries, fmt.Sprintf(
				`{"id":%d,"created":"2026-09-13T09:00:00Z","expiration":null,`+
					`"slug":"slug-%d","document":%d,"file_version":%q}`, id, id, document, version))
		}

		return strings.Join(entries, ",")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = w.Write([]byte(`{"count":102,"next":"?page=2","previous":null,"results":[` +
				pageEntries(1, 101, 7, "archive") + `]}`))
		case "2":
			_, _ = w.Write([]byte(`{"count":102,"next":null,"previous":null,"results":[` +
				`{"id":101,"created":"2026-09-13T09:30:00Z","expiration":null,"slug":"slug-101",` +
				`"document":8,"file_version":"original"},` +
				`{"id":102,"created":"2026-09-13T09:30:00Z",` +
				`"expiration":"2026-12-31T23:59:59Z","slug":"slug-102",` +
				`"document":8,"file_version":"original"}]}`))
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("page"))
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	links, err := client.ListShareLinks(t.Context())
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}

	if len(links) != 102 {
		t.Fatalf("links = %d, want 102 across two pages", len(links))
	}

	if links[0].ID.Get() != 1 || links[0].DocumentID.Get() != 7 ||
		links[0].FileVersion != ShareLinkFileVersionArchive {
		t.Fatalf("links[0] = %+v", links[0])
	}

	if !links[0].Expiration.IsZero() {
		t.Fatalf("links[0].Expiration = %v, want zero for null", links[0].Expiration)
	}

	if links[100].FileVersion != ShareLinkFileVersionOriginal {
		t.Fatalf("links[100].FileVersion = %q", links[100].FileVersion)
	}

	if links[101].Expiration.IsZero() {
		t.Fatal("links[101].Expiration must decode the server value")
	}

	if len(requestedPages) != 2 {
		t.Fatalf("requested pages = %v, want [1 2]", requestedPages)
	}
}

func TestCreateShareLinkSendsDocumentAndFileVersion(t *testing.T) {
	t.Parallel()

	var postBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		postBody = raw

		_, _ = w.Write([]byte(`{"id":9,"created":"2026-09-13T09:00:00Z","expiration":null,` +
			`"slug":"server-slug","document":7,"file_version":"original"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	link, err := client.CreateShareLink(t.Context(), CreateShareLinkRequest{
		DocumentID:  NewDocumentID(7),
		FileVersion: ShareLinkFileVersionOriginal,
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	if link.ID.Get() != 9 || link.Slug != "server-slug" || link.DocumentID.Get() != 7 {
		t.Fatalf("link = %+v", link)
	}

	if !link.Expiration.IsZero() {
		t.Fatalf("link.Expiration = %v, want zero for null", link.Expiration)
	}

	var payload map[string]any
	if err := json.Unmarshal(postBody, &payload); err != nil {
		t.Fatalf("decode POST body %q: %v", postBody, err)
	}

	if payload["document"].(float64) != 7 || payload["file_version"] != "original" {
		t.Fatalf("POST payload = %v", payload)
	}

	if _, hasSlug := payload["slug"]; hasSlug {
		t.Fatal("POST payload must not send a slug (the server generates it)")
	}
}

func TestCreateShareLinkDefaultsFileVersionAndCarriesExpiration(t *testing.T) {
	t.Parallel()

	var postBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		postBody = raw

		_, _ = w.Write([]byte(`{"id":10,"created":"2026-09-13T09:00:00Z",` +
			`"expiration":"2026-12-31T23:59:59Z","slug":"s","document":7,` +
			`"file_version":"archive"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	expiration := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	if _, err := client.CreateShareLink(t.Context(), CreateShareLinkRequest{
		DocumentID: NewDocumentID(7),
		Expiration: &expiration,
	}); err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	var payload struct {
		Document    int        `json:"document"`
		FileVersion string     `json:"file_version"`
		Expiration  *time.Time `json:"expiration"`
	}
	if err := json.Unmarshal(postBody, &payload); err != nil {
		t.Fatalf("decode POST body %q: %v", postBody, err)
	}

	if payload.FileVersion != "" {
		t.Fatalf(
			"zero FileVersion must be omitted (server defaults to archive), got %q",
			payload.FileVersion,
		)
	}

	if payload.Expiration == nil || !payload.Expiration.Equal(expiration) {
		t.Fatalf("POST expiration = %v, want %v", payload.Expiration, expiration)
	}
}

func TestCreateShareLinkRejectsZeroDocument(t *testing.T) {
	t.Parallel()

	server, requestCount := newNoRequestServer(t)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.CreateShareLink(t.Context(), CreateShareLinkRequest{}); err == nil {
		t.Fatal("expected an error for the zero document ID")
	}

	if n := requestCount(); n != 0 {
		t.Fatalf("requests = %d, want 0", n)
	}
}

func TestDeleteShareLinkSendsDelete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/share_links/4/" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.DeleteShareLink(t.Context(), NewShareLinkID(4)); err != nil {
		t.Fatalf("DeleteShareLink: %v", err)
	}
}

func TestDeleteShareLinkRejectsZeroID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("zero share-link ID must not reach the server")
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = client.DeleteShareLink(t.Context(), NewShareLinkID(0))
	if err == nil {
		t.Fatal("expected a rejection for share link ID 0")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Errorf("family = %v, want Rejection (%v)", family, err)
	}
}

func TestDeleteShareLinkSurfacesServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = client.DeleteShareLink(t.Context(), NewShareLinkID(12))
	if err == nil {
		t.Fatal("expected the server error to surface")
	}

	if !strings.Contains(err.Error(), "12") {
		t.Errorf("error should name the share link ID: %v", err)
	}
}
