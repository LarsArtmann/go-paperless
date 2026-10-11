package paperless

import id "github.com/larsartmann/go-branded-id"

// Branded identifier types for every Paperless-ngx entity the SDK
// addresses. Each ID is a distinct type at compile time (phantom brands via
// github.com/larsartmann/go-branded-id), so a TagID can never be passed
// where a CorrespondentID is expected — the class of bug raw int/string
// parameters invite. Serialization always uses the raw value (JSON numbers
// for numeric IDs, strings for task UUIDs); the zero value means "unset"
// and serializes to JSON null.

// Brand types are empty phantom markers: they exist only as type parameters
// and add zero runtime cost. They are exported so consumers (and the
// paperlesstest fake) can instantiate the same ID types in generic code;
// Name() makes IDs debug-visible ("document:42") in logs and test failures.
type (
	// TagBrand brands Paperless-ngx tag IDs.
	TagBrand struct{}
	// CorrespondentBrand brands Paperless-ngx correspondent IDs.
	CorrespondentBrand struct{}
	// DocumentTypeBrand brands Paperless-ngx document type IDs.
	DocumentTypeBrand struct{}
	// DocumentBrand brands Paperless-ngx document IDs.
	DocumentBrand struct{}
	// DocumentVersionBrand brands Paperless-ngx document version IDs
	// (the versions[] entries a document's checksum lives in since 3.x).
	DocumentVersionBrand struct{}
	// TaskBrand brands Paperless-ngx consumption task IDs (UUIDs).
	TaskBrand struct{}
	// CustomFieldBrand brands Paperless-ngx custom-field definition IDs.
	CustomFieldBrand struct{}
	// StoragePathBrand brands Paperless-ngx storage path IDs.
	StoragePathBrand struct{}
	// UserBrand brands Paperless-ngx user IDs (note authors).
	UserBrand struct{}
	// NoteBrand brands Paperless-ngx document note IDs.
	NoteBrand struct{}
	// ShareLinkBrand brands Paperless-ngx share link IDs.
	ShareLinkBrand struct{}
	// SavedViewBrand brands Paperless-ngx saved view IDs.
	SavedViewBrand struct{}
)

// Name returns the brand prefix for tag IDs in logs and test failures.
func (TagBrand) Name() string { return "tag" }

// Name returns the brand prefix for correspondent IDs in logs and test failures.
func (CorrespondentBrand) Name() string { return "correspondent" }

// Name returns the brand prefix for document type IDs in logs and test failures.
func (DocumentTypeBrand) Name() string { return "document-type" }

// Name returns the brand prefix for document IDs in logs and test failures.
func (DocumentBrand) Name() string { return "document" }

// Name returns the brand prefix for document version IDs in logs and failures.
func (DocumentVersionBrand) Name() string { return "document-version" }

// Name returns the brand prefix for task IDs in logs and test failures.
func (TaskBrand) Name() string { return "task" }

// Name returns the brand prefix for custom-field IDs in logs and test failures.
func (CustomFieldBrand) Name() string { return "custom-field" }

// Name returns the brand prefix for storage path IDs in logs and test failures.
func (StoragePathBrand) Name() string { return "storage-path" }

// Name returns the brand prefix for user IDs in logs and test failures.
func (UserBrand) Name() string { return "user" }

// Name returns the brand prefix for note IDs in logs and test failures.
func (NoteBrand) Name() string { return "note" }

// Name returns the brand prefix for share link IDs in logs and test failures.
func (ShareLinkBrand) Name() string { return "share-link" }

// Name returns the brand prefix for saved view IDs in logs and test failures.
func (SavedViewBrand) Name() string { return "saved-view" }

// TagID identifies a Paperless-ngx tag. Zero means "no tag".
type TagID = id.ID[TagBrand, int]

// CorrespondentID identifies a Paperless-ngx correspondent. Zero means
// "no correspondent" (the wire null decodes to zero).
type CorrespondentID = id.ID[CorrespondentBrand, int]

// DocumentTypeID identifies a Paperless-ngx document type. Zero means
// "no document type".
type DocumentTypeID = id.ID[DocumentTypeBrand, int]

// DocumentID identifies a Paperless-ngx document. Zero means "unknown
// document" — never a document reference. The underlying type is int64
// because consumption tasks report document IDs as full JSON integers and
// consumers (e.g. bank-sync ledgers) store them as int64.
type DocumentID = id.ID[DocumentBrand, int64]

// DocumentVersionID identifies one entry of a Paperless-ngx document's
// versions[] array.
type DocumentVersionID = id.ID[DocumentVersionBrand, int]

// TaskID identifies a Paperless-ngx asynchronous consumption task (a UUID
// string). Zero means "no task".
type TaskID = id.ID[TaskBrand, string]

// CustomFieldID identifies a Paperless-ngx custom-field definition.
type CustomFieldID = id.ID[CustomFieldBrand, int]

// StoragePathID identifies a Paperless-ngx storage path definition.
type StoragePathID = id.ID[StoragePathBrand, int]

// UserID identifies a Paperless-ngx user (e.g. a document note's author).
type UserID = id.ID[UserBrand, int]

// NoteID identifies a Paperless-ngx document note.
type NoteID = id.ID[NoteBrand, int]

// ShareLinkID identifies a Paperless-ngx share link.
type ShareLinkID = id.ID[ShareLinkBrand, int]

// SavedViewID identifies a Paperless-ngx saved view.
type SavedViewID = id.ID[SavedViewBrand, int]

// NewTagID creates a TagID from a raw Paperless-ngx tag ID.
func NewTagID(v int) TagID { return id.NewID[TagBrand](v) }

// NewCorrespondentID creates a CorrespondentID from a raw Paperless-ngx
// correspondent ID.
func NewCorrespondentID(v int) CorrespondentID { return id.NewID[CorrespondentBrand](v) }

// NewDocumentTypeID creates a DocumentTypeID from a raw Paperless-ngx
// document type ID.
func NewDocumentTypeID(v int) DocumentTypeID { return id.NewID[DocumentTypeBrand](v) }

// NewDocumentID creates a DocumentID from a raw Paperless-ngx document ID.
func NewDocumentID(v int64) DocumentID { return id.NewID[DocumentBrand](v) }

// NewDocumentVersionID creates a DocumentVersionID from a raw Paperless-ngx
// document version ID.
func NewDocumentVersionID(v int) DocumentVersionID { return id.NewID[DocumentVersionBrand](v) }

// NewTaskID creates a TaskID from a Paperless-ngx task UUID.
func NewTaskID(v string) TaskID { return id.NewID[TaskBrand](v) }

// NewCustomFieldID creates a CustomFieldID from a raw Paperless-ngx
// custom-field definition ID.
func NewCustomFieldID(v int) CustomFieldID { return id.NewID[CustomFieldBrand](v) }

// NewStoragePathID creates a StoragePathID from a raw Paperless-ngx storage
// path ID.
func NewStoragePathID(v int) StoragePathID { return id.NewID[StoragePathBrand](v) }

// NewUserID creates a UserID from a raw Paperless-ngx user ID.
func NewUserID(v int) UserID { return id.NewID[UserBrand](v) }

// NewNoteID creates a NoteID from a raw Paperless-ngx note ID.
func NewNoteID(v int) NoteID { return id.NewID[NoteBrand](v) }

// NewShareLinkID creates a ShareLinkID from a raw Paperless-ngx share link ID.
func NewShareLinkID(v int) ShareLinkID { return id.NewID[ShareLinkBrand](v) }

// NewSavedViewID creates a SavedViewID from a raw Paperless-ngx saved view ID.
func NewSavedViewID(v int) SavedViewID { return id.NewID[SavedViewBrand](v) }
