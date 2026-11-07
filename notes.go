package paperless

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// DocumentNoteUser is the (possibly anonymous) author of a document note.
//
// The public API struct stays tag-free (the exported JSON shape is a compatibility contract); the payload
// mirror stays unexported. Unification is the v2 scope.
type DocumentNoteUser struct {
	ID        UserID
	Username  string
	FirstName string
	LastName  string
}

// DocumentNote is one operator comment attached to a stored document.
//
// Embedding the shared fields would change the public struct's composite-literal surface; unify them in the
// ROADMAP v2 typed-API scope instead.
type DocumentNote struct {
	ID      NoteID
	Note    string
	Created time.Time
	// User is the note's author, or nil when the server cannot resolve it
	// (e.g. the author account was deleted).
	User *DocumentNoteUser
}

// documentNoteUserPayload mirrors the BasicUserSerializer the notes
// endpoint embeds.
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type documentNoteUserPayload struct {
	ID        UserID `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// documentNotePayload mirrors the Note serializer fields the SDK relies on.
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type documentNotePayload struct {
	ID      NoteID                   `json:"id"`
	Note    string                   `json:"note"`
	Created time.Time                `json:"created"`
	User    *documentNoteUserPayload `json:"user"`
}

// decodeDocumentNotes maps the server's note array (the notes endpoint is
// not paginated and always answers with a bare JSON array).
func decodeDocumentNotes(raw []byte) ([]DocumentNote, error) {
	var payloads []documentNotePayload
	if err := json.Unmarshal(raw, &payloads); err != nil {
		return nil, errorfamily.WrapCorruption(err, "paperless.decode_notes",
			"could not decode document notes")
	}

	notes := make([]DocumentNote, 0, len(payloads))
	for _, payload := range payloads {
		note := DocumentNote{
			ID:      payload.ID,
			Note:    payload.Note,
			Created: payload.Created,
		}

		if payload.User != nil {
			note.User = &DocumentNoteUser{
				ID:        payload.User.ID,
				Username:  payload.User.Username,
				FirstName: payload.User.FirstName,
				LastName:  payload.User.LastName,
			}
		}

		notes = append(notes, note)
	}

	return notes, nil
}

// ListDocumentNotes returns the notes attached to one document, newest
// first (the server's ordering). The notes endpoint is not paginated —
// the response is a bare JSON array.
func (c *Client) ListDocumentNotes(
	ctx context.Context,
	documentID DocumentID,
) ([]DocumentNote, error) {
	raw, err := c.doRequest(
		ctx,
		http.MethodGet,
		fmt.Sprintf(pathDocumentNotes, documentID.Get()),
		"",
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list document notes (document %d): %w",
			documentID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return decodeDocumentNotes(raw)
}

// AddDocumentNote appends one note to a document and returns the server's
// updated note list (the notes endpoint answers every mutation with the
// full remaining list, newest first).
func (c *Client) AddDocumentNote(
	ctx context.Context,
	documentID DocumentID,
	note string,
) ([]DocumentNote, error) {
	if documentID.IsZero() {
		return nil, errorfamily.NewRejection("paperless.empty_document_id",
			"document ID is required to add a note")
	}

	if note == "" {
		return nil, errorfamily.NewRejection("paperless.empty_note",
			"note text is required")
	}

	payload, err := json.Marshal(struct {
		Note string `json:"note"`
	}{Note: note})
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "paperless.marshal_note",
			"could not encode the note payload")
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		fmt.Sprintf(pathDocumentNotes, documentID.Get()),
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"add document note (document %d): %w",
			documentID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return decodeDocumentNotes(raw)
}

// DeleteDocumentNote removes one note from a document and returns the
// server's updated note list. The note is addressed by the query
// parameter the endpoint expects (DELETE .../notes/?id=<noteID>).
func (c *Client) DeleteDocumentNote(
	ctx context.Context,
	documentID DocumentID,
	noteID NoteID,
) ([]DocumentNote, error) {
	if documentID.IsZero() {
		return nil, errorfamily.NewRejection("paperless.empty_document_id",
			"document ID is required to delete a note")
	}

	if noteID.IsZero() {
		return nil, errorfamily.NewRejection("paperless.empty_note_id",
			"note ID is required to delete a note")
	}

	query := url.Values{}
	query.Set("id", strconv.Itoa(noteID.Get()))

	raw, err := c.doRequest(
		ctx,
		http.MethodDelete,
		fmt.Sprintf(pathDocumentNotes, documentID.Get()),
		query.Encode(),
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"delete document note (document %d, note %d): %w", //nolint:erraudit // keeps inner code+family
			documentID.Get(),
			noteID.Get(),
			err,
		)
	}

	return decodeDocumentNotes(raw)
}
