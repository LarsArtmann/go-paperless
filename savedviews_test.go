package paperless

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

//nolint:dupl // the cap tests are intentionally parallel per-endpoint checks
func TestListSavedViewsCapStopsAtMaxPages(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		page := r.URL.Query().Get("page")

		entries := make([]string, 0, documentListPageSize)
		for i := range documentListPageSize {
			entries = append(entries, fmt.Sprintf(
				`{"id":%d,"name":"view-%s-%03d","filter_rules":[]}`, i, page, i))
		}

		_, _ = w.Write([]byte(`{"results":[` + strings.Join(entries, ",") + `]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	views, err := client.ListSavedViews(t.Context())
	if err != nil {
		t.Fatalf("ListSavedViews: %v", err)
	}

	if len(views) != maxDocumentListPages*documentListPageSize {
		t.Fatalf("views = %d, want exactly %d (100 full pages)", len(views),
			maxDocumentListPages*documentListPageSize)
	}

	if requests != maxDocumentListPages {
		t.Fatalf("requests = %d, want exactly %d (the cap must stop the scan, not hang)",
			requests, maxDocumentListPages)
	}
}

func TestListSavedViewsMapsFilterRules(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[` +
			`{"id":5,"name":"Inbox","show_on_dashboard":true,"show_in_sidebar":false,` +
			`"sort_field":"created","sort_reverse":true,` +
			`"filter_rules":[{"rule_type":6,"value":"has_tag:1"},` +
			`{"rule_type":19,"value":"invoice"}]}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	views, err := client.ListSavedViews(t.Context())
	if err != nil {
		t.Fatalf("ListSavedViews: %v", err)
	}

	if len(views) != 1 {
		t.Fatalf("views = %d, want 1", len(views))
	}

	view := views[0]
	if view.ID.Get() != 5 || view.Name != "Inbox" || !view.ShowOnDashboard || view.ShowInSidebar {
		t.Fatalf("view = %+v", view)
	}

	if view.SortField != "created" || !view.SortReverse {
		t.Fatalf("sort = %q reverse=%t", view.SortField, view.SortReverse)
	}

	if len(view.FilterRules) != 2 || view.FilterRules[0].RuleType != SavedViewRuleTypeHasTagsAll ||
		view.FilterRules[0].Value != "has_tag:1" {
		t.Fatalf("filter rules = %+v", view.FilterRules)
	}

	if view.FilterRules[1].RuleType != SavedViewRuleType(19) {
		t.Fatalf("unknown rule type must decode unchanged, got %+v", view.FilterRules[1])
	}
}

func TestCreateSavedViewSendsStableFields(t *testing.T) {
	t.Parallel()

	var postBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		postBody = raw

		_, _ = w.Write([]byte(`{"id":6,"name":"Inbox","show_on_dashboard":true,` +
			`"show_in_sidebar":false,"sort_field":"created","sort_reverse":true,` +
			`"filter_rules":[{"rule_type":6,"value":"has_tag:1"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	id, err := client.CreateSavedView(t.Context(), CreateSavedViewRequest{
		Name:            "Inbox",
		ShowOnDashboard: true,
		SortField:       "created",
		SortReverse:     true,
		FilterRules: []SavedViewFilterRule{{
			RuleType: SavedViewRuleTypeHasTagsAll,
			Value:    "has_tag:1",
		}},
	})
	if err != nil {
		t.Fatalf("CreateSavedView: %v", err)
	}

	if id.Get() != 6 {
		t.Fatalf("id = %d, want 6", id)
	}

	var payload map[string]any
	if err := json.Unmarshal(postBody, &payload); err != nil {
		t.Fatalf("decode POST body %q: %v", postBody, err)
	}

	if payload["name"] != "Inbox" || payload["sort_field"] != "created" {
		t.Fatalf("POST payload = %v", payload)
	}

	if payload["show_on_dashboard"] != true || payload["sort_reverse"] != true {
		t.Fatalf("POST payload = %v", payload)
	}
}

func TestCreateSavedViewRejectsEmptyName(t *testing.T) {
	t.Parallel()

	server, requestCount := newNoRequestServer(t)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.CreateSavedView(t.Context(), CreateSavedViewRequest{})
	if err == nil {
		t.Fatal("expected an error for the empty name")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, err)
	}

	if n := requestCount(); n != 0 {
		t.Fatalf("requests = %d, want 0", n)
	}
}

func TestDeleteSavedViewSendsDelete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/saved_views/5/" {
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

	if err := client.DeleteSavedView(t.Context(), NewSavedViewID(5)); err != nil {
		t.Fatalf("DeleteSavedView: %v", err)
	}
}

// The delete-family rejects zero IDs without touching the server and wraps
// server failures with the object ID — the highest-traffic untested paths
// behind the coverage drop from ~90% to ~88.6%.

func TestDeleteSavedViewRejectsZeroID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("zero saved-view ID must not reach the server")
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = client.DeleteSavedView(t.Context(), NewSavedViewID(0))
	if err == nil {
		t.Fatal("expected a rejection for saved view ID 0")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Errorf("family = %v, want Rejection (%v)", family, err)
	}
}

func TestDeleteSavedViewSurfacesServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = client.DeleteSavedView(t.Context(), NewSavedViewID(12))
	if err == nil {
		t.Fatal("expected the server error to surface")
	}

	if !strings.Contains(err.Error(), "12") {
		t.Errorf("error should name the saved view ID: %v", err)
	}
}

// TestSavedViewFilterRulesNeverNil pins the normalization contract: a
// served view with an absent filter_rules field and one with an explicit
// empty array BOTH decode into a non-nil empty slice on SavedView, so
// consumers range over view.FilterRules without nil checks and compare
// "no rules" as len == 0 in every shape the server can serve.
func TestSavedViewFilterRulesNeverNil(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; version=10")

		_, _ = w.Write([]byte(`{"results": [
			{"id": 1, "name": "absent", "filter_rules": null},
			{"id": 2, "name": "empty-array", "filter_rules": []}
		], "count": 2, "next": null, "previous": null}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	views, err := client.ListSavedViews(t.Context())
	if err != nil {
		t.Fatalf("ListSavedViews: %v", err)
	}

	if len(views) != 2 {
		t.Fatalf("views = %d, want 2", len(views))
	}

	for _, view := range views {
		if view.FilterRules == nil {
			t.Errorf(
				"view %q decoded nil FilterRules — the normalization contract makes every served shape a non-nil empty slice",
				view.Name,
			)
		}

		if len(view.FilterRules) != 0 {
			t.Errorf("view %q decoded %d rules, want 0", view.Name, len(view.FilterRules))
		}
	}
}
