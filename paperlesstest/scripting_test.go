package paperlesstest

import (
	"context"
	"testing"
	"time"

	paperless "github.com/larsartmann/go-paperless"
)

func TestScriptedFailureIsATaskError(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithTaskFailure("OCR exploded"))
	client := newTestClient(t, server)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename: "broken.pdf",
		Content:  []byte("%PDF-broken"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	_, waitErr := client.WaitForTask(context.Background(), taskID, time.Millisecond)
	if waitErr == nil {
		t.Fatal("WaitForTask = nil error, want the task_failed rejection")
	}

	outcome, _, getErr := client.GetTask(context.Background(), taskID)
	if getErr != nil {
		t.Fatalf("GetTask: %v", getErr)
	}

	if outcome.ErrorMessage != "OCR exploded" {
		t.Errorf("error message = %q, want %q", outcome.ErrorMessage, "OCR exploded")
	}
}

func TestScriptedDuplicateInTrash(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithTaskDuplicate(42, true))
	client := newTestClient(t, server)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename: "dupe.pdf",
		Content:  []byte("%PDF-dupe"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	outcome, err := client.WaitForTask(context.Background(), taskID, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	duplicateID, inTrash, refused := outcome.Duplicate()
	if !refused || duplicateID != 42 || !inTrash {
		t.Errorf(
			"duplicate = (id %d, inTrash %v, refused %v), want (42, true, true)",
			duplicateID,
			inTrash,
			refused,
		)
	}
}

func TestScriptedSuccessPointsAtFixtureWithoutStoring(t *testing.T) {
	t.Parallel()

	server := NewServer(t,
		WithDocuments(Document{Title: "fixture", Content: []byte("%PDF-fixture")}),
		WithTaskSuccess(1),
	)
	client := newTestClient(t, server)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename: "pointed.pdf",
		Content:  []byte("%PDF-pointed"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	outcome, err := client.WaitForTask(context.Background(), taskID, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	if outcome.Status != paperless.TaskStatusSuccess || outcome.DocumentID != 1 {
		t.Errorf("outcome = %+v, want success pointing at fixture document 1", outcome)
	}

	// A scripted success is task-only: the fake does not store the upload,
	// so the document list still holds exactly the seeded fixture.
	if got := len(server.Documents()); got != 1 {
		t.Errorf("stored documents = %d, want 1 (scripted success must not store the upload)", got)
	}
}

func TestPendingThenSuccessStoresDocument(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithTaskPendingThenSuccess(2))
	client := newTestClient(t, server)

	taskID, err := client.Upload(context.Background(), paperless.UploadRequest{
		Filename: "slow.pdf",
		Content:  []byte("%PDF-slow"),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	outcome, err := client.WaitForTask(context.Background(), taskID, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	if outcome.Status != paperless.TaskStatusSuccess {
		t.Errorf("status = %q, want success after the pending polls", outcome.Status)
	}

	// Exactly two pending polls preceded the terminal answer (plus the
	// first immediate poll = three polls total).
	records := server.Requests()
	polls := 0

	for _, record := range records {
		if record.Path == "/api/tasks/" {
			polls++
		}
	}

	if polls < 3 {
		t.Errorf("task polls = %d, want at least 3 (initial + 2 pending)", polls)
	}

	checksums, listErr := client.ListDocumentChecksums(context.Background())
	if listErr != nil {
		t.Fatalf("ListDocumentChecksums: %v", listErr)
	}

	if _, ok := checksums[ChecksumOf([]byte("%PDF-slow"))]; !ok {
		t.Error("consumed document missing from the list after pending-then-success")
	}
}

func TestScriptTaskQueueRunsFIFO(t *testing.T) {
	t.Parallel()

	server := NewServer(t,
		WithTaskFailure("first fails"),
	)
	server.ScriptTask(TaskPlan{Kind: TaskPlanSuccess})
	client := newTestClient(t, server)
	ctx := context.Background()

	first, err := client.Upload(
		ctx,
		paperless.UploadRequest{Filename: "a.pdf", Content: []byte("%PDF-a")},
	)
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}

	if _, err := client.WaitForTask(ctx, first, time.Millisecond); err == nil {
		t.Error("first scripted task = nil error, want the scripted failure")
	}

	second, err := client.Upload(
		ctx,
		paperless.UploadRequest{Filename: "b.pdf", Content: []byte("%PDF-b")},
	)
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}

	outcome, err := client.WaitForTask(ctx, second, time.Millisecond)
	if err != nil {
		t.Fatalf("second task: %v", err)
	}

	if outcome.Status != paperless.TaskStatusSuccess {
		t.Errorf("second outcome = %+v, want success", outcome)
	}
}

func TestSeedTaskServesAFixedTaskID(t *testing.T) {
	t.Parallel()

	server := NewServer(t)
	server.SeedTask("task-from-past", TaskPlan{Kind: TaskPlanDuplicate, DocumentID: 7})

	outcome, found, err := newTestClient(t, server).GetTask(context.Background(), "task-from-past")
	if err != nil || !found {
		t.Fatalf("GetTask = (%+v, %v, %v), want the planted task", outcome, found, err)
	}

	duplicateID, _, refused := outcome.Duplicate()
	if !refused || duplicateID != 7 {
		t.Errorf("duplicate = (id %d, refused %v), want (7, true)", duplicateID, refused)
	}
}
