package paperlesstest

import (
	"context"
	"testing"
	"time"

	paperless "github.com/larsartmann/go-paperless"
)

func TestDocumentNotesRoundTrip(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithDocuments(Document{Title: "noted", Content: []byte("%PDF-noted")}))
	client := newTestClient(t, server)
	ctx := context.Background()

	notes, err := client.ListDocumentNotes(ctx, 1)
	if err != nil {
		t.Fatalf("ListDocumentNotes: %v", err)
	}

	if len(notes) != 0 {
		t.Fatalf("fresh document notes = %d, want 0", len(notes))
	}

	notes, err = client.AddDocumentNote(ctx, 1, "first")
	if err != nil {
		t.Fatalf("AddDocumentNote: %v", err)
	}

	notes, err = client.AddDocumentNote(ctx, 1, "second")
	if err != nil {
		t.Fatalf("AddDocumentNote second: %v", err)
	}

	if len(notes) != 2 || notes[0].Note != "second" {
		t.Fatalf("notes after adds = %+v, want newest first", notes)
	}

	if notes[0].User == nil || notes[0].User.Username == "" {
		t.Errorf("note user = %+v, want the fake's default author", notes[0].User)
	}

	if server.NoteCount(1) != 2 {
		t.Errorf("fake store note count = %d, want 2", server.NoteCount(1))
	}

	remaining, err := client.DeleteDocumentNote(ctx, 1, notes[1].ID)
	if err != nil {
		t.Fatalf("DeleteDocumentNote: %v", err)
	}

	if len(remaining) != 1 || remaining[0].Note != "second" {
		t.Errorf("remaining notes = %+v, want only the newest note", remaining)
	}
}

func TestNotesOnUnknownDocumentAnswer404(t *testing.T) {
	t.Parallel()

	server := NewServer(t)

	if _, err := newTestClient(t, server).ListDocumentNotes(context.Background(), 99); err == nil {
		t.Error("ListDocumentNotes on unknown document = nil error, want a client rejection")
	}
}

func TestShareLinksRoundTrip(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithDocuments(Document{Title: "shared", Content: []byte("%PDF-shared")}))
	client := newTestClient(t, server)
	ctx := context.Background()

	expiration := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	link, err := client.CreateShareLink(ctx, paperless.CreateShareLinkRequest{
		DocumentID:  1,
		FileVersion: paperless.ShareLinkFileVersionOriginal,
		Expiration:  &expiration,
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	if link.ID != 1 || link.Slug == "" || link.DocumentID != 1 {
		t.Fatalf("created link = %+v, want id 1, slug set, document 1", link)
	}

	if !link.Expiration.Equal(expiration) {
		t.Errorf("expiration = %v, want %v", link.Expiration, expiration)
	}

	defaultLink, err := client.CreateShareLink(ctx, paperless.CreateShareLinkRequest{DocumentID: 1})
	if err != nil {
		t.Fatalf("CreateShareLink default: %v", err)
	}

	if defaultLink.FileVersion != paperless.ShareLinkFileVersionArchive {
		t.Errorf(
			"default file version = %q, want archive (the server default)",
			defaultLink.FileVersion,
		)
	}

	links, err := client.ListShareLinks(ctx)
	if err != nil {
		t.Fatalf("ListShareLinks: %v", err)
	}

	if len(links) != 2 {
		t.Fatalf("links = %d, want 2", len(links))
	}

	if err := client.DeleteShareLink(ctx, link.ID); err != nil {
		t.Fatalf("DeleteShareLink: %v", err)
	}

	links, err = client.ListShareLinks(ctx)
	if err != nil {
		t.Fatalf("ListShareLinks after delete: %v", err)
	}

	if len(links) != 1 || links[0].ID != defaultLink.ID {
		t.Errorf("links after delete = %+v, want only the surviving link", links)
	}
}

func TestSavedViewsRoundTrip(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	client := newTestClient(t, server)
	ctx := context.Background()

	created, err := client.CreateSavedView(ctx, paperless.CreateSavedViewRequest{
		Name:            "Inbox",
		ShowOnDashboard: true,
		ShowInSidebar:   true,
		SortField:       "created",
		SortReverse:     true,
		FilterRules: []paperless.SavedViewFilterRule{
			{RuleType: paperless.SavedViewRuleTypeHasTagsAll, Value: "1"},
			{RuleType: paperless.SavedViewRuleTypeContent, Value: "invoice"},
		},
	})
	if err != nil {
		t.Fatalf("CreateSavedView: %v", err)
	}

	if created != 1 {
		t.Errorf("created id = %d, want 1", created)
	}

	views, err := client.ListSavedViews(ctx)
	if err != nil {
		t.Fatalf("ListSavedViews: %v", err)
	}

	if len(views) != 1 {
		t.Fatalf("views = %d, want 1", len(views))
	}

	view := views[0]
	if view.Name != "Inbox" || !view.ShowOnDashboard || view.SortField != "created" {
		t.Errorf("view = %+v, want the stored fields preserved", view)
	}

	if len(view.FilterRules) != 2 ||
		view.FilterRules[0].RuleType != paperless.SavedViewRuleTypeHasTagsAll ||
		view.FilterRules[1].Value != "invoice" {
		t.Errorf("filter rules = %+v, want both rules preserved in order", view.FilterRules)
	}
}

func TestProbeCapabilitiesEchoesConfiguredAPIVersion(t *testing.T) {
	t.Parallel()

	server := NewServer(t,
		WithAPIVersion("42"),
		WithDocuments(
			Document{Title: "flat", Checksum: "c1"},
			Document{Title: "mixed", Checksum: "c2"},
		),
	)

	probe, err := newTestClient(t, server).ProbeCapabilities(context.Background())
	if err != nil {
		t.Fatalf("ProbeCapabilities: %v", err)
	}

	if probe.AcceptAPIVersion != "42" {
		t.Errorf("negotiated version = %q, want 42", probe.AcceptAPIVersion)
	}

	if probe.DocumentsSampled != 2 || !probe.FlatChecksum {
		t.Errorf("probe = %+v, want 2 sampled flat-checksum documents", probe)
	}
}
