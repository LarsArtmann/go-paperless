package paperless

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// UploadRequest describes one document to hand to Paperless-ngx's consumer.
//
// Embedding the shared fields would change the public struct's composite-literal surface; unify them in the
// ROADMAP v2 typed-API scope instead.
type UploadRequest struct {
	// Filename is the original file name (e.g. "invoice.pdf"). Paperless-ngx
	// derives the stored document name and archive serial numbering from it.
	Filename string
	// Content holds the raw file bytes. Gmail attachments are capped at 25 MB,
	// so buffering in memory is safe.
	Content []byte
	// Title overrides the consumer-assigned title when set.
	Title string
	// Created sets the document's creation date (typically the email date).
	Created time.Time
	// CorrespondentID is the Paperless-ngx correspondent (e.g. the email
	// sender) assigned on consumption. Zero leaves the field unset.
	CorrespondentID CorrespondentID
	// TagIDs are Paperless-ngx tag IDs applied on consumption.
	TagIDs []TagID
	// DocumentTypeID is the Paperless-ngx document type (e.g. "Email")
	// applied on consumption. Zero leaves the type unset.
	DocumentTypeID DocumentTypeID
	// CustomFields are custom-field values applied on consumption, e.g. the
	// provenance field carrying the Gmail message ID. Nil leaves them unset.
	CustomFields []CustomFieldValue
}

// CustomFieldValue assigns a value to one custom field by its definition ID.
//
// The public API struct stays tag-free (the exported JSON shape is a compatibility contract); the payload
// mirror stays unexported. Unification is the v2 scope.
type CustomFieldValue struct {
	Field CustomFieldID
	Value string
}

// Upload posts a document to the consumption queue. It returns the task ID
// (UUID) Paperless-ngx assigned; consumption itself is asynchronous — the
// caller can poll `/api/tasks/?task_id=<id>` for the resulting document.
func (c *Client) Upload(ctx context.Context, req UploadRequest) (TaskID, error) {
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	filePart, err := writer.CreateFormFile("document", req.Filename)
	if err != nil {
		return TaskID{}, errorfamily.WrapInfrastructure(err, "paperless.build_multipart", "could not build upload form").
			WithContext("filename", req.Filename)
	}

	if _, writeErr := filePart.Write(req.Content); writeErr != nil {
		return TaskID{}, errorfamily.WrapInfrastructure(
			writeErr,
			"paperless.write_document",
			"could not write document bytes",
		).WithContext("filename", req.Filename)
	}

	if metaErr := writeUploadMetadata(writer, req); metaErr != nil {
		return TaskID{}, metaErr
	}

	if closeErr := writer.Close(); closeErr != nil {
		return TaskID{}, errorfamily.WrapInfrastructure(
			closeErr,
			"paperless.close_multipart",
			"could not finalize upload form",
		)
	}

	taskIDBytes, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathUploadDocument,
		"",
		&body,
		writer.FormDataContentType(),
	)
	if err != nil {
		return TaskID{}, fmt.Errorf(
			"upload %q: %w",
			req.Filename,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	taskID := NewTaskID(strings.Trim(strings.TrimSpace(string(taskIDBytes)), `"'`))
	if taskID.IsZero() {
		return TaskID{}, errorfamily.NewCorruption("paperless.missing_task_id",
			"Paperless-ngx accepted the upload but returned no task ID").
			WithContext("filename", req.Filename)
	}

	return taskID, nil
}

// writeUploadMetadata adds the optional metadata form fields shared by all uploads.
func writeUploadMetadata(writer *multipart.Writer, req UploadRequest) error {
	if req.Title != "" {
		if err := writer.WriteField("title", req.Title); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_title",
				"could not write title field",
			)
		}
	}

	if !req.Created.IsZero() {
		if err := writer.WriteField("created", req.Created.Format(time.DateOnly)); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_created",
				"could not write created field",
			)
		}
	}

	if !req.CorrespondentID.IsZero() {
		if err := writer.WriteField(
			"correspondent",
			strconv.Itoa(req.CorrespondentID.Get()),
		); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_correspondent",
				"could not write correspondent field",
			)
		}
	}

	for _, tagID := range req.TagIDs {
		if err := writer.WriteField("tags", strconv.Itoa(tagID.Get())); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_tags",
				"could not write tags field",
			).WithContext("tag_id", strconv.Itoa(tagID.Get()))
		}
	}

	if !req.DocumentTypeID.IsZero() {
		if err := writer.WriteField(
			"document_type",
			strconv.Itoa(req.DocumentTypeID.Get()),
		); err != nil {
			return errorfamily.WrapInfrastructure(
				err,
				"paperless.write_document_type",
				"could not write document_type field",
			)
		}
	}

	return writeCustomFieldsField(writer, req.CustomFields)
}

// writeCustomFieldsField serializes the custom-field values into the JSON
// form field Paperless-ngx's consumption endpoint expects.
func writeCustomFieldsField(writer *multipart.Writer, fields []CustomFieldValue) error {
	if len(fields) == 0 {
		return nil
	}

	params := make([]customFieldValuePayload, 0, len(fields))
	for _, field := range fields {
		params = append(params, customFieldValuePayload(field))
	}

	encoded, err := json.Marshal(params)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"paperless.write_custom_fields",
			"could not encode custom_fields field",
		)
	}

	if err := writer.WriteField("custom_fields", string(encoded)); err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"paperless.write_custom_fields",
			"could not write custom_fields field",
		)
	}

	return nil
}
