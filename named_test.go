package paperless

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

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

	if id.Get() != 42 {
		t.Fatalf("id = %d, want 42", id)
	}
}

func TestEnsureTagSelfHealsLegacyAutoTag(t *testing.T) {
	t.Parallel()

	patched := make(chan *namedPayload[TagBrand], 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags/":
			// A tag created by an older InboxClean release: matching_algorithm
			// "auto" (6), the exact shape the 2026-09 classifier incident left
			// behind on live installs.
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"id":42,"name":"gmail","matching_algorithm":6}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/tags/42/":
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read patch body: %v", err)
			}

			// The demote PATCH must carry ONLY matching_algorithm: DRF rejects
			// a PATCH that re-sends a blank required name with 400 Bad Request
			// (the 2026-09-17 live failure: "Bad Request: /api/tags/1/" on
			// every sync tick, so the auto tag never healed).
			if bytes.Contains(raw, []byte(`"name"`)) {
				t.Errorf("patch body re-sent the name field: %s", raw)
			}

			var payload namedPayload[TagBrand]
			if err := json.UnmarshalRead(bytes.NewReader(raw), &payload); err != nil {
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

	if id.Get() != 42 {
		t.Fatalf("id = %d, want 42", id)
	}

	select {
	case payload := <-patched:
		if payload.MatchingAlgorithm != int(matchingAlgorithmNone) {
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

	if id.Get() != 9 {
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
			var payload namedPayload[TagBrand]
			if err := json.UnmarshalRead(r.Body, &payload); err != nil {
				t.Errorf("decode create body: %v", err)
			}

			if payload.Name != "newtag" {
				t.Errorf("created name = %q", payload.Name)
			}

			// Tags are provenance metadata: they must never join the
			// classifier's training set (auto matching would leak the tag onto
			// similar-looking manual scans).
			if payload.MatchingAlgorithm != int(matchingAlgorithmNone) {
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

	if id.Get() != 7 {
		t.Fatalf("id = %d, want 7", id)
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

	if id.Get() != 7 {
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
				MatchingAlgorithm int    `json:"matching_algorithm"`
			}

			if err := json.UnmarshalRead(r.Body, &payload); err != nil {
				t.Errorf("decode payload: %v", err)
			}

			if payload.Name != "Alior Bank" {
				t.Errorf("created name = %q", payload.Name)
			}

			if payload.MatchingAlgorithm != int(matchingAlgorithmAuto) {
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

	if id.Get() != 12 {
		t.Errorf("id = %d, want 12", id)
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

	name, err := client.GetCorrespondentName(t.Context(), NewCorrespondentID(7))
	if err != nil {
		t.Fatalf("GetCorrespondentName(7): %v", err)
	}

	if name != "Alior Bank" {
		t.Fatalf("name = %q, want %q", name, "Alior Bank")
	}

	if requestedPath != "/api/correspondents/7/" {
		t.Fatalf("request path = %q, want the detail endpoint", requestedPath)
	}

	if _, err := client.GetCorrespondentName(t.Context(), NewCorrespondentID(99)); err == nil {
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

	id, err := client.EnsureDocumentType(t.Context(), "Email")
	if err != nil {
		t.Fatalf("EnsureDocumentType: %v", err)
	}

	if id.Get() != 3 {
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

	name, err := client.GetDocumentTypeName(t.Context(), NewDocumentTypeID(5))
	if err != nil {
		t.Fatalf("GetDocumentTypeName(5): %v", err)
	}

	if name != "Invoice" {
		t.Fatalf("name = %q, want Invoice", name)
	}

	if _, err := client.GetDocumentTypeName(t.Context(), NewDocumentTypeID(9)); err == nil {
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
					Name     string `json:"name"`
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

	id, err := client.EnsureCustomField(t.Context(), "gmail_message_id")
	if err != nil {
		t.Fatalf("EnsureCustomField: %v", err)
	}

	if id.Get() != 1 {
		t.Fatalf("first ensure id = %d, want 1", id)
	}

	id, err = client.EnsureCustomField(t.Context(), "gmail_message_id")
	if err != nil {
		t.Fatalf("second EnsureCustomField: %v", err)
	}

	if id.Get() != 1 {
		t.Fatalf("second ensure id = %d, want the same field 1", id)
	}

	found, exists, err := client.FindCustomField(t.Context(), "gmail_message_id")
	if err != nil || !exists || found.Get() != 1 {
		t.Fatalf("FindCustomField = (%d, %v, %v), want (1, true, nil)", found, exists, err)
	}

	_, exists, err = client.FindCustomField(t.Context(), "missing_field")
	if err != nil || exists {
		t.Fatalf("FindCustomField missing = (%d, %v, %v), want (0, false, nil)", 0, exists, err)
	}
}

func TestEnsureStoragePathCreatesWhenMissing(t *testing.T) {
	t.Parallel()

	var postBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if got := r.URL.Query().Get("name__iexact"); got != "Invoices" {
				t.Errorf("name__iexact = %q, want Invoices", got)
			}

			if got := r.URL.Query().Get("page_size"); got != "1" {
				t.Errorf("page_size = %q, want 1", got)
			}

			_, _ = w.Write([]byte(`{"results":[]}`))
		case http.MethodPost:
			raw, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				t.Errorf("read POST body: %v", readErr)
			}

			postBody = raw
			_, _ = w.Write(
				[]byte(`{"id":5,"slug":"invoices","name":"Invoices","path":"{created_year}/"}`),
			)
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureStoragePath(t.Context(), "Invoices", "{created_year}/")
	if err != nil {
		t.Fatalf("EnsureStoragePath: %v", err)
	}

	if id.Get() != 5 {
		t.Fatalf("id = %d, want 5", id)
	}

	var payload struct {
		Name string  `json:"name"`
		Path string  `json:"path"`
		Slug *string `json:"slug"`
	}
	if err := json.Unmarshal(postBody, &payload); err != nil {
		t.Fatalf("decode POST body %q: %v", postBody, err)
	}

	if payload.Name != "Invoices" || payload.Path != "{created_year}/" {
		t.Fatalf("POST payload = %+v, want name + directory template", payload)
	}

	if payload.Slug != nil {
		t.Fatalf(
			"POST payload must not send a slug (the server generates it), got %q",
			*payload.Slug,
		)
	}
}

func TestEnsureStoragePathKeepsExistingTemplateWithoutPOST(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf(
				"unexpected %s: an existing storage path must be reused, not recreated",
				r.Method,
			)
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		_, _ = w.Write([]byte(`{"results":[{"id":3,"slug":"archive","name":"Archive",` +
			`"path":"{correspondent}/"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.EnsureStoragePath(t.Context(), "Archive", "{created_year}/{title}/")
	if err != nil {
		t.Fatalf("EnsureStoragePath: %v", err)
	}

	if id.Get() != 3 {
		t.Fatalf("id = %d, want the existing 3", id)
	}
}

func TestEnsureStoragePathRejectsEmptyArgs(t *testing.T) {
	t.Parallel()

	server, requestCount := newNoRequestServer(t)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, emptyNameErr := client.EnsureStoragePath(t.Context(), "", "path")
	if emptyNameErr == nil {
		t.Fatal("expected an error for the empty name")
	}

	if family := errorfamily.Classify(emptyNameErr); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, emptyNameErr)
	}

	if _, err := client.EnsureStoragePath(t.Context(), "Name", ""); err == nil {
		t.Fatal("expected an error for the empty directory template")
	}

	if n := requestCount(); n != 0 {
		t.Fatalf("requests = %d, want 0 (validation happens before any HTTP)", n)
	}
}

func TestFindStoragePathReturnsFoundFlag(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("name__iexact"); got == "Archive" {
			_, _ = w.Write([]byte(`{"results":[{"id":3,"slug":"archive","name":"Archive",` +
				`"path":"{correspondent}/"}]}`))

			return
		}

		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, found, err := client.FindStoragePath(t.Context(), "Archive")
	if err != nil || !found || id.Get() != 3 {
		t.Fatalf("FindStoragePath(Archive) = (%d, %t, %v), want (3, true, nil)", id, found, err)
	}

	id, found, err = client.FindStoragePath(t.Context(), "Missing")
	if err != nil || found || id.Get() != 0 {
		t.Fatalf("FindStoragePath(Missing) = (%d, %t, %v), want (0, false, nil)", id, found, err)
	}
}

func TestListStoragePathsPaginatesAndMaps(t *testing.T) {
	t.Parallel()

	var requestedPages []string

	entries := make([]string, 0, 100)
	for i := range 100 {
		entries = append(entries, fmt.Sprintf(
			`{"id":%d,"slug":"path-%d","name":"Path %d","path":"dir-%d/"}`, i, i, i, i))
	}

	fullPage := `{"count":101,"next":"?page=2","previous":null,"results":[` +
		strings.Join(entries, ",") + `]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPages = append(requestedPages, r.URL.Query().Get("page"))

		if got := r.URL.Query().Get("page_size"); got != "100" {
			t.Errorf("page_size = %q, want 100", got)
		}

		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = w.Write([]byte(fullPage))
		case "2":
			_, _ = w.Write([]byte(`{"count":101,"next":null,"previous":null,"results":[` +
				`{"id":100,"slug":"path-100","name":"Path 100","path":"dir-100/"}]}`))
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

	paths, err := client.ListStoragePaths(t.Context())
	if err != nil {
		t.Fatalf("ListStoragePaths: %v", err)
	}

	if len(paths) != 101 {
		t.Fatalf("paths = %d, want 101 across two pages", len(paths))
	}

	if paths[0].ID.Get() != 0 || paths[0].Name != "Path 0" || paths[0].Slug != "path-0" ||
		paths[0].Path != "dir-0/" {
		t.Fatalf("paths[0] = %+v, want the mapped first entry", paths[0])
	}

	if paths[100].ID.Get() != 100 || paths[100].Name != "Path 100" {
		t.Fatalf("paths[100] = %+v, want the mapped second-page entry", paths[100])
	}

	if len(requestedPages) != 2 || requestedPages[0] != "1" || requestedPages[1] != "2" {
		t.Fatalf("requested pages = %v, want [1 2]", requestedPages)
	}
}

func TestListStoragePathsCapStopsAtMaxPages(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		page := r.URL.Query().Get("page")

		entries := make([]string, 0, documentListPageSize)
		for i := range documentListPageSize {
			entries = append(entries, fmt.Sprintf(
				`{"id":%d,"name":"path-%s-%03d","path":"/docs/%s/%03d"}`, i, page, i, page, i))
		}

		_, _ = w.Write([]byte(`{"results":[` + strings.Join(entries, ",") + `]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	paths, err := client.ListStoragePaths(t.Context())
	if err != nil {
		t.Fatalf("ListStoragePaths: %v", err)
	}

	if len(paths) != maxDocumentListPages*documentListPageSize {
		t.Fatalf("paths = %d, want exactly %d (100 full pages)", len(paths),
			maxDocumentListPages*documentListPageSize)
	}

	if requests != maxDocumentListPages {
		t.Fatalf("requests = %d, want exactly %d (the cap must stop the scan, not hang)",
			requests, maxDocumentListPages)
	}
}

// The self-heal PATCH failure must surface with the tag name — the one
// updateMatchingAlgorithm branch no other test drives.
func TestEnsureTagSelfHealPatchFailureSurfaces(t *testing.T) {
	t.Parallel()

	var patchRequests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(
				`{"results":[{"id":4,"name":"legacy","matching_algorithm":6}]}`,
			))
		case http.MethodPatch:
			patchRequests++

			w.WriteHeader(http.StatusInternalServerError)

			return
		default:
			t.Errorf("unexpected method %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.EnsureTag(t.Context(), "legacy")
	if err == nil {
		t.Fatal("expected the failed self-heal PATCH to surface")
	}

	if patchRequests != 1 {
		t.Errorf("PATCH requests = %d, want exactly 1", patchRequests)
	}
}

// errPartWriteFailed is the sentinel failingPartWriter always returns.
