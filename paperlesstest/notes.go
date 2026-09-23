package paperlesstest

import (
	"encoding/json/v2"
	"net/http"
	"strconv"
	"time"
)

// noteEntity is one note attached to a stored document.
type noteEntity struct {
	ID      int
	Note    string
	Created time.Time
}

// noteWire mirrors the Note serializer fields the SDK relies on; the fake
// attaches a fixed author (the real server embeds the acting user).
type noteWire struct {
	ID      int           `json:"id"`
	Note    string        `json:"note"`
	Created time.Time     `json:"created"`
	User    *noteUserWire `json:"user"`
}

// noteUserWire mirrors the BasicUserSerializer the notes endpoint embeds.
type noteUserWire struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// defaultNoteUser is the fixed author the fake attaches to every note.
var defaultNoteUser = &noteUserWire{
	ID:        1,
	Username:  "paperlesstest",
	FirstName: "Paperless",
	LastName:  "Test",
}

// handleDocumentNotesLocked serves one document's notes routes (the notes
// endpoint is not paginated and answers every mutation with the full
// remaining list, newest first). It reports whether the request matched.
// The caller must hold s.mu.
func (s *Server) handleDocumentNotesLocked(w http.ResponseWriter, r *http.Request, id int) bool {
	switch r.Method {
	case http.MethodGet:
		if s.findDocumentLocked(id) == nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

			return true
		}

		s.writeNotes(w, id)

		return true
	case http.MethodPost:
		var payload struct {
			Note string `json:"note"`
		}
		if err := json.Unmarshal(readBody(r), &payload); err != nil {
			s.t.Errorf("paperlesstest: decode note create body: %v", err)
			s.writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{detailKey: "invalid note payload"},
			)

			return true
		}

		if payload.Note == "" {
			s.writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{detailKey: "note text is required"},
			)

			return true
		}

		if s.findDocumentLocked(id) == nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

			return true
		}

		s.nextNoteID++
		s.notes[id] = append(s.notes[id], noteEntity{
			ID:      s.nextNoteID,
			Note:    payload.Note,
			Created: time.Now().UTC(),
		})

		s.writeNotes(w, id)

		return true
	case http.MethodDelete:
		noteID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil || noteID <= 0 {
			s.writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{detailKey: "the id query parameter is required"},
			)

			return true
		}

		remaining := make([]noteEntity, 0, len(s.notes[id]))
		for _, note := range s.notes[id] {
			if note.ID != noteID {
				remaining = append(remaining, note)
			}
		}

		if len(remaining) == len(s.notes[id]) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

			return true
		}

		s.notes[id] = remaining

		s.writeNotes(w, id)

		return true
	default:
		s.methodNotAllowed(w, r, http.MethodGet, http.MethodPost, http.MethodDelete)

		return true
	}
}

// writeNotes serves one document's notes as the bare JSON array the notes
// endpoint answers with (newest first, the server's ordering). The caller
// must hold s.mu.
func (s *Server) writeNotes(w http.ResponseWriter, id int) {
	notes := s.notes[id]

	wires := make([]noteWire, 0, len(notes))
	for index := len(notes) - 1; index >= 0; index-- {
		wires = append(wires, noteWire{
			ID:      notes[index].ID,
			Note:    notes[index].Note,
			Created: notes[index].Created,
			User:    defaultNoteUser,
		})
	}

	s.writeJSON(w, http.StatusOK, wires)
}

// NoteCount reports how many notes one document carries (a test assertion
// helper; the SDK has no note-count read API).
func (s *Server) NoteCount(id int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.notes[id])
}
