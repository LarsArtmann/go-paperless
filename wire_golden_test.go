package paperless

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// goldenBodyServer captures the raw body of the first request matching
// method and path suffix; the returned read function hands the bytes to the
// golden assertions. The responder keeps the call alive with whatever the
// caller needs (task id, empty object).
func goldenBodyServer(t *testing.T, method, pathSuffix, respond string) (*Client, func() string) {
	t.Helper()

	var captured string

	var once bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !once && r.Method == method && strings.HasSuffix(r.URL.Path, pathSuffix) {
			raw, _ := io.ReadAll(r.Body)

			captured = string(raw)
			once = true
		}

		w.Header().Set("Content-Type", "application/json; version=10")

		if respond != "" {
			_, _ = w.Write([]byte(respond))
		}
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return client, func() string { return captured }
}

// The wire-shape goldens pin the exact JSON bytes the SDK sends for the
// contracts consumers and the paperlesstest fake depend on: zero-value
// branded IDs marshal as null (ADR 0004), FK clearing rides explicit
// nulls through pointer-to-zero fields, and an empty saved-view filter
// rule set stays an empty array rather than vanishing. A wire-shape drift
// breaks these tests before it can break a consumer's archive.

func TestGoldenNamedCreateBodySendsNullID(t *testing.T) {
	t.Parallel()

	client, body := goldenBodyServer(
		t,
		http.MethodPost,
		"/api/tags/",
		`{"id":1,"name":"provenance","matching_algorithm":0,"slug":"provenance"}`,
	)

	tagID, err := client.EnsureTag(t.Context(), "provenance")
	if err != nil {
		t.Fatalf("EnsureTag: %v", err)
	}

	if tagID.IsZero() {
		t.Fatal("EnsureTag returned a zero ID")
	}

	const want = `{"id":null,"name":"provenance","matching_algorithm":0}`

	if got := body(); got != want {
		t.Fatalf("create body = %s, want golden %s", got, want)
	}
}

func TestGoldenUploadCustomFieldsPartSendsFieldAndValue(t *testing.T) {
	t.Parallel()

	var contentType string

	var rawBody []byte

	var once bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !once && r.Method == http.MethodPost &&
			strings.HasSuffix(r.URL.Path, "/api/documents/post_document/") {
			rawBody, _ = io.ReadAll(r.Body)

			contentType = r.Header.Get("Content-Type")
			once = true
		}

		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(`"task-golden"`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Upload(t.Context(), UploadRequest{
		Filename: "statement.pdf",
		Content:  []byte("%PDF-1.4 golden"),
		CustomFields: []CustomFieldValue{
			{Field: NewCustomFieldID(7), Value: "msg-1"},
		},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("parse multipart content type %q: %v", contentType, err)
	}

	reader := multipart.NewReader(bytes.NewReader(rawBody), params["boundary"])

	var customFields string

	for {
		part, partErr := reader.NextPart()
		if partErr != nil {
			break
		}

		if part.FormName() == "custom_fields" {
			value, _ := io.ReadAll(part)

			customFields = string(value)

			break
		}
	}

	const want = `[{"field":7,"value":"msg-1"}]`

	if customFields != want {
		t.Fatalf("custom_fields part = %s, want golden %s", customFields, want)
	}
}

func TestGoldenFKClearingPatchSendsExplicitNulls(t *testing.T) {
	t.Parallel()

	client, body := goldenBodyServer(t, http.MethodPatch, "/api/documents/42/", `{}`)

	zeroCorrespondent := NewCorrespondentID(0)
	zeroDocumentType := NewDocumentTypeID(0)

	err := client.UpdateDocument(t.Context(), NewDocumentID(42), UpdateDocumentRequest{
		CorrespondentID: &zeroCorrespondent,
		DocumentTypeID:  &zeroDocumentType,
	})
	if err != nil {
		t.Fatalf("UpdateDocument: %v", err)
	}

	const want = `{"correspondent":null,"document_type":null}`

	if got := body(); got != want {
		t.Fatalf("patch body = %s, want golden %s", got, want)
	}
}

func TestGoldenSavedViewCreateBodyKeepsEmptyFilterRules(t *testing.T) {
	t.Parallel()

	client, body := goldenBodyServer(t, http.MethodPost, "/api/saved_views/", `{"id":3}`)

	viewID, err := client.CreateSavedView(t.Context(), CreateSavedViewRequest{
		Name:            "inbox",
		ShowOnDashboard: true,
	})
	if err != nil {
		t.Fatalf("CreateSavedView: %v", err)
	}

	if viewID.IsZero() {
		t.Fatal("CreateSavedView returned a zero ID")
	}

	const want = `{"name":"inbox","show_on_dashboard":true,"show_in_sidebar":false,"sort_field":"","sort_reverse":false,"filter_rules":[]}`

	if got := body(); got != want {
		t.Fatalf("create body = %s, want golden %s", got, want)
	}
}

// TestGoldenBodiesAreStableJSON double-checks the goldens themselves: every
// pinned body must round-trip as valid JSON, so a malformed golden fails
// here instead of silently pinning garbage.
func TestGoldenBodiesAreStableJSON(t *testing.T) {
	t.Parallel()

	for name, golden := range map[string]string{
		"named create":      `{"id":null,"name":"provenance","matching_algorithm":0}`,
		"fk-clearing patch": `{"correspondent":null,"document_type":null}`,
		"saved view create": `{"name":"inbox","show_on_dashboard":true,"show_in_sidebar":false,"sort_field":"","sort_reverse":false,"filter_rules":[]}`,
	} {
		var decoded any

		if err := json.Unmarshal([]byte(golden), &decoded); err != nil {
			t.Errorf("golden %s is not valid JSON: %v", name, err)
		}

		if !strings.Contains(golden, "null") && name != "saved view create" {
			t.Errorf("golden %s lost its null-contract marker: %s", name, golden)
		}
	}

	if errorfamily.Classify(nil) != errorfamily.Rejection {
		t.Error("errorfamily contract drifted: nil must classify Rejection")
	}
}
