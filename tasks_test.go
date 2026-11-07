package paperless

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

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

	outcome, found, err := client.GetTask(t.Context(), NewTaskID("task-uuid-1"))
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if !found {
		t.Fatal("expected the task to be found")
	}

	if outcome.Status != TaskStatusSuccess || !outcome.Status.Terminal() {
		t.Fatalf("status = %q, want terminal success", outcome.Status)
	}

	if outcome.DocumentID.Get() != 42 {
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

	outcome, found, err := client.GetTask(t.Context(), NewTaskID("task-uuid-2"))
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

	if outcome.DocumentID.Get() != 7 {
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

	outcome, found, err := client.GetTask(t.Context(), NewTaskID("task-uuid-3"))
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

	if outcome.DocumentID.Get() != 0 {
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

	outcome, found, err := client.GetTask(t.Context(), NewTaskID("task-uuid-4"))
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

	_, found, err := client.GetTask(t.Context(), NewTaskID("missing-uuid"))
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

	outcome, found, err := client.GetTask(t.Context(), NewTaskID("task-uuid-5"))
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if !found {
		t.Fatal("expected the task to be found")
	}

	if outcome.Status != TaskStatusSuccess || outcome.DocumentID.Get() != 9 {
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

	_, _, err = client.GetTask(t.Context(), NewTaskID("task-uuid-6"))
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
}

func TestWaitForTaskPendingThenSuccess(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		if gotQuery := r.URL.Query().Get("task_id"); gotQuery != "task-uuid-1" {
			t.Errorf("task_id = %q, want task-uuid-1", gotQuery)
		}

		if requests == 1 {
			_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-1","status":"pending"}]}`))

			return
		}

		_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-1","status":"success",` +
			`"result_data":{"document_id":42},"related_document_ids":[42]}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, err := client.WaitForTask(t.Context(), NewTaskID("task-uuid-1"), time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	if outcome.Status != TaskStatusSuccess {
		t.Fatalf("status = %q, want success", outcome.Status)
	}

	if outcome.DocumentID.Get() != 42 {
		t.Fatalf("DocumentID = %d, want 42", outcome.DocumentID)
	}

	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (pending then success)", requests)
	}
}

func TestWaitForTaskToleratesNotFoundThenSuccess(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			_, _ = w.Write([]byte(`{"results":[]}`))

			return
		}

		_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-2","status":"success",` +
			`"result_data":{"document_id":7}}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, err := client.WaitForTask(t.Context(), NewTaskID("task-uuid-2"), time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	if outcome.DocumentID.Get() != 7 {
		t.Fatalf("DocumentID = %d, want 7", outcome.DocumentID)
	}

	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (not-found keeps polling)", requests)
	}
}

func TestWaitForTaskTerminalFailureIsRejection(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-3","status":"failure",` +
			`"result_data":{"error_type":"ParseError","error_message":"bad pdf"}}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, err := client.WaitForTask(t.Context(), NewTaskID("task-uuid-3"), time.Millisecond)
	if err == nil {
		t.Fatal("expected the terminal failure to surface as an error")
	}

	if requests != 1 {
		t.Fatalf("requests = %d, want 1 (terminal on first poll)", requests)
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family for paperless.task_failed, got %v (%v)", family, err)
	}

	if outcome.Status != TaskStatusFailure || outcome.ErrorMessage != "bad pdf" {
		t.Fatalf("outcome = %+v, want populated failure outcome", outcome)
	}
}

func TestWaitForTaskDuplicateRefusalIsNotAnError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-4","status":"failure",` +
			`"result_data":{"duplicate_of":7,"duplicate_in_trash":false}}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, err := client.WaitForTask(t.Context(), NewTaskID("task-uuid-4"), time.Millisecond)
	if err != nil {
		t.Fatalf("duplicate refusal must not be an error, got %v", err)
	}

	documentID, inTrash, refused := outcome.Duplicate()
	if !refused || documentID.Get() != 7 || inTrash {
		t.Fatalf("Duplicate() = (%d, %t, %t), want (7, false, true)", documentID, inTrash, refused)
	}
}

func TestWaitForTaskContextDeadlineSurfacesLastPollError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()

	_, err = client.WaitForTask(ctx, NewTaskID("task-uuid-5"), 5*time.Millisecond)
	if err == nil {
		t.Fatal("expected the context deadline to surface as an error")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Infrastructure {
		t.Fatalf("expected Infrastructure family for the abandoned poll, got %v (%v)", family, err)
	}

	if !strings.Contains(err.Error(), "last poll error") {
		t.Fatalf("error must surface the last poll error, got %v", err)
	}
}

// newNoRequestServer starts a server that only counts received requests;
// rejection tests assert the counter stays zero to prove the rejected call
// never touched the wire. The server closes via t.Cleanup.

// newNoRequestServer starts a server that only counts received requests;
// rejection tests assert the counter stays zero to prove the rejected call
// never touched the wire. The server closes via t.Cleanup.
func newNoRequestServer(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		requests++
	}))
	t.Cleanup(server.Close)

	return server, func() int { return requests }
}

func TestWaitForTaskRejectsEmptyTaskID(t *testing.T) {
	t.Parallel()

	server, requestCount := newNoRequestServer(t)

	client, err := New(server.URL, "token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.WaitForTask(t.Context(), NewTaskID(""), time.Millisecond)
	if err == nil {
		t.Fatal("expected an error for the empty task ID")
	}

	if family := errorfamily.Classify(err); family != errorfamily.Rejection {
		t.Fatalf("expected Rejection family, got %v (%v)", family, err)
	}

	if n := requestCount(); n != 0 {
		t.Fatalf("requests = %d, want 0 (validation happens before any HTTP)", n)
	}
}

func TestWaitForTaskWithRetryRecoversFromTransientPollFailure(t *testing.T) {
	t.Parallel()

	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		switch requests {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			_, _ = w.Write([]byte(`{"results":[]}`))
		default:
			_, _ = w.Write([]byte(`{"results":[{"task_id":"task-uuid-6","status":"success",` +
				`"result_data":{"document_id":42}}]}`))
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token", WithRetry(RetryPolicy{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond,
		MaxDelay:     2 * time.Millisecond,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	outcome, err := client.WaitForTask(t.Context(), NewTaskID("task-uuid-6"), time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}

	if outcome.DocumentID.Get() != 42 {
		t.Fatalf("DocumentID = %d, want 42", outcome.DocumentID)
	}

	if requests != 3 {
		t.Fatalf("requests = %d, want 3 (retried 503, tolerated not-found, success)", requests)
	}
}

func TestTaskOutcomeDuplicateReportsRefusalDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		outcome     TaskOutcome
		wantID      DocumentID
		wantInTrash bool
		wantRefused bool
	}{
		{
			name:    "not a refusal",
			outcome: TaskOutcome{Status: TaskStatusSuccess, DocumentID: NewDocumentID(42)},
		},
		{
			name:        "refused duplicate",
			outcome:     TaskOutcome{DuplicateRefused: true, DocumentID: NewDocumentID(7)},
			wantID:      NewDocumentID(7),
			wantRefused: true,
		},
		{
			name: "refused duplicate in trash",
			outcome: TaskOutcome{
				DuplicateRefused: true,
				DuplicateInTrash: true,
				DocumentID:       NewDocumentID(9),
			},
			wantID:      NewDocumentID(9),
			wantInTrash: true,
			wantRefused: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			documentID, inTrash, refused := tt.outcome.Duplicate()
			if documentID != tt.wantID || inTrash != tt.wantInTrash || refused != tt.wantRefused {
				t.Fatalf("Duplicate() = (%d, %t, %t), want (%d, %t, %t)",
					documentID, inTrash, refused, tt.wantID, tt.wantInTrash, tt.wantRefused)
			}
		})
	}
}
