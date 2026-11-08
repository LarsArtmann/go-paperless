package paperless

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClassifyTaskEdgeMix exercises classifyTask through GetTask across
// the status x result-data combinations that singular tests miss:
// unknown status strings (lowercased and kept, never a panic), failure
// WITH related_document_ids fallback, mixed result_data shapes, and
// whitespace/case tolerance.
func TestClassifyTaskEdgeMix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		body                 string
		wantStatus           TaskStatus
		wantDocumentID       int64
		wantDuplicate        bool
		wantDuplicateInTrash bool
		wantErrorMessage     string
	}{
		{
			name:       "unknown status lowercased and surfaced verbatim",
			body:       `{"results": [{"task_id": "t", "status": "  WEIRD  ", "result_data": {}}]}`,
			wantStatus: TaskStatus("weird"),
		},
		{
			name:           "failure falls back to related_document_ids",
			body:           `{"results": [{"task_id": "t", "status": "FAILURE", "related_document_ids": [9, 8]}]}`,
			wantStatus:     TaskStatusFailure,
			wantDocumentID: 9,
		},
		{
			name:             "duplicate_refused with document_id wins over duplicate_of",
			body:             `{"results": [{"task_id": "t", "status": "FAILURE", "result_data": {"document_id": 3, "duplicate_of": 5, "duplicate_refused": true, "duplicate_in_trash": true, "error_message": "dup"}}]}`,
			wantStatus:       TaskStatusFailure,
			wantDocumentID:   3,
			wantErrorMessage: "dup",
		},
		{
			name:       "success with no result_data at all",
			body:       `{"results": [{"task_id": "t", "status": "success"}]}`,
			wantStatus: TaskStatusSuccess,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json; version=10")

					_, _ = w.Write([]byte(testCase.body))
				}),
			)
			defer server.Close()

			client, err := New(server.URL, "token")
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			outcome, found, err := client.GetTask(t.Context(), NewTaskID("t"))
			if err != nil {
				t.Fatalf("GetTask: %v", err)
			}

			if !found {
				t.Fatal("task not found")
			}

			if outcome.Status != testCase.wantStatus {
				t.Errorf("status = %q, want %q", outcome.Status, testCase.wantStatus)
			}

			if testCase.wantDocumentID != 0 && outcome.DocumentID.Get() != testCase.wantDocumentID {
				t.Errorf(
					"document id = %d, want %d",
					outcome.DocumentID.Get(),
					testCase.wantDocumentID,
				)
			}

			if outcome.DuplicateRefused != testCase.wantDuplicate {
				t.Errorf(
					"duplicate refused = %v, want %v",
					outcome.DuplicateRefused,
					testCase.wantDuplicate,
				)
			}

			if outcome.ErrorMessage != testCase.wantErrorMessage {
				t.Errorf(
					"error message = %q, want %q",
					outcome.ErrorMessage,
					testCase.wantErrorMessage,
				)
			}
		})
	}
}
