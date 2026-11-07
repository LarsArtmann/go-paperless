package paperless

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// TaskStatus is a Paperless-ngx consumption task's lifecycle state. The v10
// API serves lowercase values; terminal states are success and failure
// (verified against paperless-ngx 3.x PaperlessTask.Status).
type TaskStatus string

// Task consumption statuses served by the v10 API (lowercase values).
const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusStarted TaskStatus = "started"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailure TaskStatus = "failure"
)

// Terminal reports whether the status is a final task state. Unknown status
// strings (future server versions) poll forever — the safe default for a
// caller deciding whether to retry.
func (s TaskStatus) Terminal() bool {
	return s == TaskStatusSuccess || s == TaskStatusFailure
}

// TaskOutcome is the classified result of one consumption-task poll.
type TaskOutcome struct {
	Status TaskStatus
	// DocumentID is the server document this task points at: the newly
	// created document after a successful consumption, or the pre-existing
	// duplicate when the server refused the upload. Zero when unknown.
	DocumentID DocumentID
	// DuplicateRefused is true when the consumer rejected the upload because
	// an identical document already exists (checksum dedup). The ledger uses
	// this to separate honest refusals from real failures.
	DuplicateRefused bool
	// DuplicateInTrash refines DuplicateRefused: the duplicate document is
	// in the trash (server treats trash duplicates as re-consumable).
	DuplicateInTrash bool
	// ErrorMessage is the server's failure reason for non-duplicate failures.
	ErrorMessage string
}

// Duplicate reports the duplicate-refusal details of an outcome. refused is
// true when the consumer rejected the upload because an identical document
// already exists (checksum dedup); documentID is that pre-existing duplicate
// and inTrash says whether it sits in the trash (the server treats trash
// duplicates as re-consumable). For non-refusals all returns are zero.
func (o TaskOutcome) Duplicate() (DocumentID, bool, bool) {
	if !o.DuplicateRefused {
		return DocumentID{}, false, false
	}

	return o.DocumentID, o.DuplicateInTrash, true
}

// taskResultData mirrors the free-form result_data dict Paperless-ngx stores
// on a finished task: {"document_id": N} after a successful consumption,
// {"duplicate_of": N, "duplicate_in_trash": bool} for refused duplicates
// (the postrun handler demotes such tasks to failure), and
// {"error_type", "error_message"} for real failures. Pointers distinguish
// absent keys from zero values.
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type taskResultData struct {
	DocumentID       *DocumentID `json:"document_id"`
	DuplicateOf      *DocumentID `json:"duplicate_of"`
	DuplicateInTrash bool        `json:"duplicate_in_trash"`
	ErrorMessage     *string     `json:"error_message"`
}

// taskPayload mirrors the Paperless-ngx v10 task object (TaskSerializerV10).
type taskPayload struct {
	TaskID             TaskID         `json:"task_id"`
	Status             string         `json:"status"`
	ResultData         taskResultData `json:"result_data"`
	RelatedDocumentIDs []DocumentID   `json:"related_document_ids"`
}

// classifyTask turns the server's task payload into the classified outcome.
// DocumentID prefers result_data.document_id, falls back to duplicate_of,
// then to the first related_document_ids entry (the server's projection of
// exactly those two fields).
func classifyTask(payload taskPayload) TaskOutcome {
	status := TaskStatus(strings.ToLower(strings.TrimSpace(payload.Status)))
	outcome := TaskOutcome{Status: status}

	switch {
	case payload.ResultData.DocumentID != nil:
		outcome.DocumentID = *payload.ResultData.DocumentID
	case payload.ResultData.DuplicateOf != nil:
		outcome.DocumentID = *payload.ResultData.DuplicateOf
		outcome.DuplicateRefused = true
		outcome.DuplicateInTrash = payload.ResultData.DuplicateInTrash
	case len(payload.RelatedDocumentIDs) > 0:
		outcome.DocumentID = payload.RelatedDocumentIDs[0]
	}

	if payload.ResultData.ErrorMessage != nil {
		outcome.ErrorMessage = *payload.ResultData.ErrorMessage
	}

	return outcome
}

// GetTask fetches one consumption task by ID (the UUID Upload returns).
// found is false when the server has no such task — the record may not be
// persisted yet (polling callers retry) or was pruned.
//
// The response is expected in the v10 paginated envelope
// (`{"results": [...]}`); a bare JSON array is tolerated defensively for
// deployments that predate pagination.
func (c *Client) GetTask(ctx context.Context, taskID TaskID) (TaskOutcome, bool, error) {
	query := url.Values{}
	query.Set("task_id", taskID.Get())

	raw, reqErr := c.doRequest(ctx, http.MethodGet, pathTasks, query.Encode(), nil, "")
	if reqErr != nil {
		return TaskOutcome{}, false, fmt.Errorf(
			"get task %s: %w",
			taskID.Get(),
			reqErr,
		) //nolint:erraudit // keeps inner code+family
	}

	var envelope struct {
		Results []taskPayload `json:"results"`
	}

	if unmarshalErr := json.Unmarshal(raw, &envelope); unmarshalErr != nil {
		// Tolerate a bare JSON array too (defensive, mirrors checksumFrom):
		// the v10 endpoint paginates, but older deployments may not. The
		// reported error wraps the ENVELOPE failure, not the array attempt,
		// because the envelope shape is what v10 promises — the array shape
		// is a fallback, and its failure adds no diagnostic value. The
		// second Unmarshal reallocates envelope.Results; negligible at a
		// one-element payload.
		envelope.Results = nil

		if arrayErr := json.Unmarshal(raw, &envelope.Results); arrayErr != nil {
			return TaskOutcome{}, false, errorfamily.WrapCorruption(
				unmarshalErr,
				"paperless.decode_task",
				"could not decode task poll result",
			).WithContext("task_id", taskID.Get())
		}
	}

	if len(envelope.Results) == 0 {
		return TaskOutcome{}, false, nil
	}

	return classifyTask(envelope.Results[0]), true, nil
}

// DefaultTaskPollInterval is the pause between polls used by WaitForTask when
// the caller passes a non-positive interval.
const DefaultTaskPollInterval = 2 * time.Second

// WaitForTask polls one consumption task until it reaches a terminal state
// (success or failure), the context is done, or the server stops answering.
//
// Polling starts immediately; interval is the pause between polls (a
// non-positive interval means DefaultTaskPollInterval). Bounds come from the
// context: pass ctx with a deadline/timeout to cap total wait time.
//
// The task may not be visible yet right after Upload returns (the server
// persists it asynchronously) — that case keeps polling. Transient poll
// failures (network blips, decode hiccups) also keep polling; the most
// recent error surfaces only when the context expires.
//
// On a terminal outcome the outcome is fully populated. err is non-nil only
// for a real terminal failure (status failure without a duplicate refusal —
// duplicate refusals are honest outcomes, not errors).
func (c *Client) WaitForTask(
	ctx context.Context,
	taskID TaskID,
	interval time.Duration,
) (TaskOutcome, error) {
	if taskID.IsZero() {
		return TaskOutcome{}, errorfamily.Newf(errorfamily.Rejection, "paperless.empty_task_id",
			"task ID is required to poll a consumption task, got %q", taskID.Get())
	}

	if interval <= 0 {
		interval = DefaultTaskPollInterval
	}

	var lastErr error

	for {
		outcome, found, err := c.GetTask(ctx, taskID)
		if err != nil {
			lastErr = err
		} else if found && outcome.Status.Terminal() {
			if outcome.Status == TaskStatusFailure && !outcome.DuplicateRefused {
				return outcome, errorfamily.NewRejection("paperless.task_failed",
					"Paperless-ngx consumption task failed").
					WithContext("task_id", taskID.Get()).
					WithContextAny("error_message", outcome.ErrorMessage)
			}

			return outcome, nil
		}

		select {
		case <-ctx.Done():
			reason := errorfamily.WrapInfrastructure(ctx.Err(),
				"paperless.task_poll_abandoned",
				"gave up waiting for the consumption task",
			).WithContext("task_id", taskID.Get())

			if lastErr != nil {
				return TaskOutcome{}, fmt.Errorf(
					"wait for task %s: %w (last poll error: %w)",
					taskID.Get(),
					reason,
					lastErr,
				) //nolint:erraudit // dual %w keeps both coded chains reachable
			}

			return TaskOutcome{}, reason
		case <-time.After(interval):
		}
	}
}
