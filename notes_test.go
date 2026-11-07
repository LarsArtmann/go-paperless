package paperless

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestListDocumentNotesReturnsBareArray(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/documents/7/notes/" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}

		_, _ = w.Write([]byte(`[` +
			`{"id":1,"note":"first","created":"2026-09-13T10:00:00Z",` +
			`"user":{"id":2,"username":"lars","first_name":"Lars","last_name":"A"}},` +
			`{"id":2,"note":"second","created":"2026-09-13T11:00:00Z","user":null}]`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	notes, err := client.ListDocumentNotes(t.Context(), NewDocumentID(7))
	if err != nil {
		t.Fatalf("ListDocumentNotes: %v", err)
	}

	if len(notes) != 2 {
		t.Fatalf("notes = %d, want 2", len(notes))
	}

	if notes[0].ID.Get() != 1 || notes[0].Note != "first" {
		t.Fatalf("notes[0] = %+v", notes[0])
	}

	if notes[0].User == nil || notes[0].User.Username != "lars" {
		t.Fatalf("notes[0].User = %+v, want the decoded user", notes[0].User)
	}

	if !notes[0].Created.Equal(time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("notes[0].Created = %v", notes[0].Created)
	}

	if notes[1].User != nil {
		t.Fatalf("notes[1].User = %+v, want nil for a null user", notes[1].User)
	}
}

func TestAddDocumentNoteSendsNoteFieldAndReturnsUpdated(t *testing.T) {
	t.Parallel()

	var postBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		raw, _ := io.ReadAll(r.Body)
		postBody = raw

		_, _ = w.Write([]byte(`[{"id":1,"note":"hello","created":"2026-09-13T12:00:00Z"}]`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	notes, err := client.AddDocumentNote(t.Context(), NewDocumentID(7), "hello")
	if err != nil {
		t.Fatalf("AddDocumentNote: %v", err)
	}

	if len(notes) != 1 || notes[0].Note != "hello" {
		t.Fatalf("notes = %+v, want the server's updated list", notes)
	}

	var payload struct {
		Note string `json:"note"`
	}
	if err := json.Unmarshal(postBody, &payload); err != nil {
		t.Fatalf("decode POST body %q: %v", postBody, err)
	}

	if payload.Note != "hello" {
		t.Fatalf("POST note = %q", payload.Note)
	}
}

func TestAddDocumentNoteRejectsEmptyNote(t *testing.T) {
	t.Parallel()

	server, requestCount := newNoRequestServer(t)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.AddDocumentNote(t.Context(), NewDocumentID(0), "x"); err == nil {
		t.Fatal("expected an error for the zero document ID")
	}

	_, err = client.AddDocumentNote(t.Context(), NewDocumentID(7), "")
	if err == nil {
		t.Fatal("expected an error for the empty note")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, err)
	}

	if n := requestCount(); n != 0 {
		t.Fatalf("requests = %d, want 0 (validation happens before any HTTP)", n)
	}
}
