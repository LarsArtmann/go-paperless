package paperlesstest

import (
	"context"
	"testing"
)

// storagePathTemplate reads one storage path's template straight from the
// fake's store (the SDK has no read API for the template itself).
func storagePathTemplate(tb testing.TB, server *Server, name string) string {
	tb.Helper()

	server.mu.Lock()
	defer server.mu.Unlock()

	for _, path := range server.storagePaths {
		if path.Name == name {
			return path.Path
		}
	}

	return ""
}

func TestEnsureTagCreatesFindsAndSelfHeals(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	client := newTestClient(t, server)
	ctx := context.Background()

	id, err := client.EnsureTag(ctx, "gmail")
	if err != nil {
		t.Fatalf("EnsureTag create: %v", err)
	}

	if id != 1 {
		t.Errorf("created tag id = %d, want 1", id)
	}

	again, err := client.EnsureTag(ctx, "gmail")
	if err != nil {
		t.Fatalf("EnsureTag find: %v", err)
	}

	if again != id {
		t.Errorf("EnsureTag again = %d, want the existing %d", again, id)
	}

	legacy := server.AddTag("legacy-auto", 6)

	healed, err := client.EnsureTag(ctx, "legacy-auto")
	if err != nil {
		t.Fatalf("EnsureTag self-heal: %v", err)
	}

	if healed != legacy {
		t.Errorf("self-heal id = %d, want the existing %d", healed, legacy)
	}

	name, err := client.GetCorrespondentName(
		ctx,
		legacy,
	) // different family, just exercising detail 404 path
	if err == nil {
		t.Errorf("correspondent lookup on a tag id = %q, want an error", name)
	}
}

func TestEnsureCorrespondentAndDocumentType(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithCorrespondents("Acme Corp"), WithDocumentTypes("Email"))
	client := newTestClient(t, server)
	ctx := context.Background()

	correspondentID, err := client.EnsureCorrespondent(ctx, "Acme Corp")
	if err != nil {
		t.Fatalf("EnsureCorrespondent existing: %v", err)
	}

	if correspondentID != 1 {
		t.Errorf("correspondent id = %d, want 1", correspondentID)
	}

	createdCorrespondent, err := client.EnsureCorrespondent(ctx, "New Sender")
	if err != nil {
		t.Fatalf("EnsureCorrespondent create: %v", err)
	}

	if createdCorrespondent != 2 {
		t.Errorf("new correspondent id = %d, want 2", createdCorrespondent)
	}

	typeID, err := client.EnsureDocumentType(ctx, "Email")
	if err != nil {
		t.Fatalf("EnsureDocumentType: %v", err)
	}

	if typeID != 1 {
		t.Errorf("document type id = %d, want 1", typeID)
	}

	senderName, err := client.GetCorrespondentName(ctx, correspondentID)
	if err != nil {
		t.Fatalf("GetCorrespondentName: %v", err)
	}

	if senderName != "Acme Corp" {
		t.Errorf("correspondent name = %q, want Acme Corp", senderName)
	}

	typeName, err := client.GetDocumentTypeName(ctx, typeID)
	if err != nil {
		t.Fatalf("GetDocumentTypeName: %v", err)
	}

	if typeName != "Email" {
		t.Errorf("document type name = %q, want Email", typeName)
	}
}

func TestFindAndEnsureCustomField(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithCustomFields("gmail-message-id"))
	client := newTestClient(t, server)
	ctx := context.Background()

	found, ok, err := client.FindCustomField(ctx, "gmail-message-id")
	if err != nil || !ok {
		t.Fatalf("FindCustomField = (%d, %v, %v), want found", found, ok, err)
	}

	if found != 1 {
		t.Errorf("found id = %d, want 1", found)
	}

	missing, ok, err := client.FindCustomField(ctx, "missing")
	if err != nil {
		t.Fatalf("FindCustomField missing: %v", err)
	}

	if ok || missing != 0 {
		t.Errorf("FindCustomField missing = (%d, %v), want (0, false)", missing, ok)
	}

	ensured, err := client.EnsureCustomField(ctx, "provenance")
	if err != nil {
		t.Fatalf("EnsureCustomField: %v", err)
	}

	if ensured != 2 {
		t.Errorf("ensured id = %d, want 2 (created after the seeded field)", ensured)
	}
}

func TestFindEnsureAndListStoragePaths(t *testing.T) {
	t.Parallel()

	server := NewServer(
		t,
		WithStoragePaths(StoragePathFixture{Name: "Invoices", Path: "{created_year}/invoices"}),
	)
	client := newTestClient(t, server)
	ctx := context.Background()

	found, ok, err := client.FindStoragePath(ctx, "invoices")
	if err != nil || !ok {
		t.Fatalf("FindStoragePath case-insensitive = (%d, %v, %v), want found", found, ok, err)
	}

	ensured, err := client.EnsureStoragePath(ctx, "Receipts", "{created_year}/receipts")
	if err != nil {
		t.Fatalf("EnsureStoragePath create: %v", err)
	}

	if ensured != 2 {
		t.Errorf("ensured id = %d, want 2", ensured)
	}

	if _, err := client.EnsureStoragePath(ctx, "Invoices", "other"); err != nil {
		t.Fatalf("EnsureStoragePath existing: %v", err)
	}

	stored := storagePathTemplate(t, server, "Invoices")
	if stored != "{created_year}/invoices" {
		t.Errorf("existing path template = %q, want the original template kept", stored)
	}

	paths, err := client.ListStoragePaths(ctx)
	if err != nil {
		t.Fatalf("ListStoragePaths: %v", err)
	}

	if len(paths) != 2 || paths[0].Slug == "" {
		t.Errorf("paths = %+v, want 2 entries with slugs", paths)
	}
}
