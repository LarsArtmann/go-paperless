package paperlesstest

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Upload is the parsed capture of one post_document multipart upload:
// every form field the SDK can send plus the file content.
type Upload struct {
	// Filename is the uploaded file's name (multipart filename).
	Filename string
	// Content is the uploaded file's bytes.
	Content []byte
	// Title is the raw "title" form field ("" when not sent).
	Title string
	// Created is the raw "created" form field (date-only, "" when not sent).
	Created string
	// Correspondent is the raw "correspondent" form field ("" when not sent).
	Correspondent string
	// DocumentType is the raw "document_type" form field ("" when not sent).
	DocumentType string
	// Tags are the raw "tags" form field values, one entry per repeated field.
	Tags []string
	// CustomFields is the raw "custom_fields" form field (JSON array, ""
	// when not sent).
	CustomFields string
}

// Uploads returns a copy of every captured upload, oldest first.
func (s *Server) Uploads() []Upload {
	s.mu.Lock()
	defer s.mu.Unlock()

	uploads := make([]Upload, 0, len(s.uploads))
	for _, upload := range s.uploads {
		copied := upload
		copied.Content = slices.Clone(upload.Content)
		copied.Tags = slices.Clone(upload.Tags)
		uploads = append(uploads, copied)
	}

	return uploads
}

// TaskPlanKind selects the terminal outcome of a scripted consumption task.
type TaskPlanKind int

const (
	// TaskPlanSuccess resolves the task as consumed.
	TaskPlanSuccess TaskPlanKind = iota
	// TaskPlanFailure resolves the task as failed (a real failure, not a
	// duplicate refusal).
	TaskPlanFailure
	// TaskPlanDuplicate resolves the task as a duplicate refusal (the
	// server's postrun handler demotes these to failure status with a
	// duplicate_of result).
	TaskPlanDuplicate
)

// TaskPlan scripts the outcome of one upload's consumption task. The fake
// pops plans first-in-first-out as uploads arrive; once the script runs
// dry, uploads take the natural path (duplicate checksum refused, anything
// else consumed).
type TaskPlan struct {
	Kind TaskPlanKind
	// DocumentID is the document the task points at. For success: the
	// fixture document reported as consumed; zero means the upload is
	// consumed naturally (the fake stores it and reports the new ID).
	// For duplicate: the pre-existing document the upload duplicates.
	DocumentID int64
	// Message is the failure reason served for TaskPlanFailure.
	Message string
	// InTrash refines TaskPlanDuplicate: the duplicate sits in the trash.
	InTrash bool
	// PendingPolls is how many task polls answer "pending" before the
	// terminal outcome is served.
	PendingPolls int
}

// WithTaskSuccess scripts the next upload's task to resolve as consumed,
// pointing at the fixture document documentID.
func WithTaskSuccess(documentID int) Option {
	return func(s *Server) {
		s.scriptTask(TaskPlan{Kind: TaskPlanSuccess, DocumentID: int64(documentID)})
	}
}

// WithTaskFailure scripts the next upload's task to fail with the given
// reason.
func WithTaskFailure(message string) Option {
	return func(s *Server) {
		s.scriptTask(TaskPlan{Kind: TaskPlanFailure, Message: message})
	}
}

// WithTaskDuplicate scripts the next upload's task as a duplicate refusal
// pointing at the pre-existing document documentID (inTrash marks a
// duplicate sitting in the trash).
func WithTaskDuplicate(documentID int, inTrash bool) Option {
	return func(s *Server) {
		s.scriptTask(
			TaskPlan{Kind: TaskPlanDuplicate, DocumentID: int64(documentID), InTrash: inTrash},
		)
	}
}

// WithTaskPendingThenSuccess scripts the next upload's task to answer
// pendingPolls polls with "pending" before being consumed naturally (the
// upload is stored and the new document reported).
func WithTaskPendingThenSuccess(pendingPolls int) Option {
	return func(s *Server) {
		s.scriptTask(TaskPlan{Kind: TaskPlanSuccess, PendingPolls: pendingPolls})
	}
}

// ScriptTask queues one task outcome for the next upload, the runtime
// equivalent of the WithTask* options (usable mid-test, after construction).
func (s *Server) ScriptTask(plan TaskPlan) {
	s.scriptTask(plan)
}

func (s *Server) scriptTask(plan TaskPlan) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.script = append(s.script, plan)
}

// taskRecord is one consumption task the fake has issued.
type taskRecord struct {
	ID string
	// plan is the resolved outcome this task terminates with.
	plan TaskPlan
	// polls counts the task polls served (drives PendingPolls).
	polls int
}

// taskWire mirrors the Paperless-ngx v10 task object (TaskSerializerV10)
// subset the SDK decodes.
type taskWire struct {
	TaskID             string         `json:"task_id"`
	Status             string         `json:"status"`
	ResultData         taskResultWire `json:"result_data"`
	RelatedDocumentIDs []int64        `json:"related_document_ids"`
}

// taskResultWire mirrors the free-form result_data dict: {"document_id": N}
// after successful consumption, {"duplicate_of": N, "duplicate_in_trash":
// B} for refused duplicates, {"error_message": "..."} for real failures.
type taskResultWire struct {
	DocumentID       *int64  `json:"document_id,omitempty"`
	DuplicateOf      *int64  `json:"duplicate_of,omitempty"`
	DuplicateInTrash bool    `json:"duplicate_in_trash"`
	ErrorMessage     *string `json:"error_message,omitempty"`
}

// routeUpload serves the post_document upload endpoint. It reports whether
// the request matched the route.
func (s *Server) routeUpload(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/api/documents/post_document/" {
		return false
	}

	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, r, http.MethodPost)

		return true
	}

	s.handleUpload(w, r)

	return true
}

// handleUpload parses the multipart upload, captures it, resolves the
// task plan (scripted first, natural otherwise), and answers with the new
// task's UUID as a bare JSON string.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadFormMemory); err != nil {
		s.t.Errorf("paperlesstest: parse upload multipart: %v", err)
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid multipart form"})

		return
	}

	file, header, err := r.FormFile("document")
	if err != nil {
		s.t.Errorf("paperlesstest: upload without document file: %v", err)
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "missing document file"})

		return
	}

	content, readErr := io.ReadAll(file)
	_ = file.Close()
	if readErr != nil {
		s.t.Errorf("paperlesstest: read uploaded document: %v", readErr)
		s.writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{"detail": "unreadable document file"},
		)

		return
	}

	upload := Upload{
		Filename:      header.Filename,
		Content:       content,
		Title:         r.FormValue("title"),
		Created:       r.FormValue("created"),
		Correspondent: r.FormValue("correspondent"),
		DocumentType:  r.FormValue("document_type"),
		Tags:          r.MultipartForm.Value["tags"],
		CustomFields:  r.FormValue("custom_fields"),
	}

	s.mu.Lock()

	s.uploads = append(s.uploads, upload)
	task := s.createTaskLocked(upload)

	s.mu.Unlock()

	s.writeJSON(w, http.StatusOK, task.ID)
}

// maxUploadFormMemory bounds the multipart parsing buffer; test uploads are
// small, so 32 MiB (the net/http default suggestion) is generous.
const maxUploadFormMemory = 32 << 20

// createTaskLocked issues the next task record for an upload: the queued
// script plan when one is pending, the natural outcome otherwise. Natural
// consumption stores the uploaded document. The caller must hold s.mu.
func (s *Server) createTaskLocked(upload Upload) *taskRecord {
	s.nextTaskNumber++

	record := &taskRecord{
		ID:   fmt.Sprintf("task-%d", s.nextTaskNumber),
		plan: TaskPlan{Kind: TaskPlanSuccess},
	}

	if scripted := s.popScriptLocked(); scripted != nil {
		record.plan = *scripted
	}

	if record.plan.DocumentID == 0 && record.plan.Kind == TaskPlanSuccess {
		record.plan = s.naturalPlanLocked(upload)
	}

	s.tasks = append(s.tasks, record)

	return record
}

// naturalPlanLocked resolves one upload through the natural consumption
// path: a duplicate checksum is refused (the plan points at the pre-existing
// document), anything else is consumed — the upload becomes a stored
// document carrying its form metadata. The caller must hold s.mu.
func (s *Server) naturalPlanLocked(upload Upload) TaskPlan {
	checksum := ChecksumOf(upload.Content)

	for _, doc := range s.documents {
		if doc.Checksum == checksum {
			return TaskPlan{Kind: TaskPlanDuplicate, DocumentID: int64(doc.ID)}
		}
	}

	stored := s.appendDocumentLocked(Document{
		Title:          consumptionTitle(upload),
		Content:        upload.Content,
		Checksum:       checksum,
		Created:        parseUploadDate(upload.Created),
		Correspondent:  atoiOrZero(upload.Correspondent),
		TagIDs:         atoiSlice(upload.Tags),
		DocumentTypeID: atoiOrZero(upload.DocumentType),
		CustomFields:   parseCustomFieldsJSON(upload.CustomFields),
	})

	return TaskPlan{Kind: TaskPlanSuccess, DocumentID: int64(stored.ID)}
}

// popScriptLocked removes and returns the next scripted plan, or nil when
// the script has run dry. The caller must hold s.mu.
func (s *Server) popScriptLocked() *TaskPlan {
	if len(s.script) == 0 {
		return nil
	}

	plan := s.script[0]
	s.script = s.script[1:]

	return &plan
}

// routeTasks serves the consumption-task queue. It reports whether the
// request matched the route.
func (s *Server) routeTasks(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/api/tasks/" {
		return false
	}

	if r.Method != http.MethodGet {
		s.methodNotAllowed(w, r, http.MethodGet)

		return true
	}

	taskID := r.URL.Query().Get("task_id")

	s.mu.Lock()
	defer s.mu.Unlock()

	results := []taskWire{}
	if payload := s.taskPayloadLocked(taskID); payload != nil {
		results = append(results, *payload)
	}

	s.writeJSON(w, http.StatusOK, envelope{Count: len(results), Results: results})

	return true
}

// taskPayloadLocked serves one task's current state for the task_id filter.
// Unknown tasks answer empty (the client keeps polling). The caller must
// hold s.mu.
func (s *Server) taskPayloadLocked(taskID string) *taskWire {
	for _, task := range s.tasks {
		if task.ID != taskID {
			continue
		}

		task.polls++

		if task.plan.PendingPolls >= task.polls {
			return &taskWire{TaskID: task.ID, Status: "pending"}
		}

		return s.terminalTaskWireLocked(task)
	}

	return nil
}

// terminalTaskWireLocked serializes one task's terminal outcome. The caller
// must hold s.mu.
func (s *Server) terminalTaskWireLocked(task *taskRecord) *taskWire {
	documentID := task.plan.DocumentID

	switch task.plan.Kind {
	case TaskPlanSuccess:
		return &taskWire{
			TaskID:             task.ID,
			Status:             "success",
			ResultData:         taskResultWire{DocumentID: &documentID},
			RelatedDocumentIDs: []int64{documentID},
		}
	case TaskPlanDuplicate:
		return &taskWire{
			TaskID: task.ID,
			Status: "failure",
			ResultData: taskResultWire{
				DuplicateOf:      &documentID,
				DuplicateInTrash: task.plan.InTrash,
			},
		}
	case TaskPlanFailure:
		message := task.plan.Message

		return &taskWire{
			TaskID:     task.ID,
			Status:     "failure",
			ResultData: taskResultWire{ErrorMessage: &message},
		}
	}

	return nil
}

// consumptionTitle derives the stored title: the upload's title field when
// present, else the filename minus its extension (the real server's
// behavior).
func consumptionTitle(upload Upload) string {
	if upload.Title != "" {
		return upload.Title
	}

	base := upload.Filename
	if slash := strings.LastIndexByte(base, '/'); slash >= 0 {
		base = base[slash+1:]
	}

	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}

	return base
}

// parseUploadDate parses the date-only created form field the SDK sends;
// unparseable values store the zero time.
func parseUploadDate(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}

	parsed, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

// atoiOrZero parses a decimal form field; garbage stores as 0 (the real
// server would reject the upload outright, which the malformed-payload
// faults cover more honestly).
func atoiOrZero(raw string) int {
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	return parsed
}

// atoiSlice parses repeated decimal form fields into IDs.
func atoiSlice(raw []string) []int {
	if len(raw) == 0 {
		return nil
	}

	ids := make([]int, 0, len(raw))
	for _, value := range raw {
		ids = append(ids, atoiOrZero(value))
	}

	return ids
}

// parseCustomFieldsJSON parses the custom_fields form field (a JSON array
// of {"field": N, "value": "..."} objects).
func parseCustomFieldsJSON(raw string) []CustomFieldValue {
	if raw == "" {
		return nil
	}

	var fields []CustomFieldValue
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil
	}

	return fields
}
