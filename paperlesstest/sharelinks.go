package paperlesstest

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// shareLinkEntity is one share link stored on the fake.
type shareLinkEntity struct {
	ID          int
	Created     time.Time
	Expiration  *time.Time
	Slug        string
	Document    int
	FileVersion string
}

// shareLinkWire mirrors the ShareLink serializer fields the SDK relies on.
type shareLinkWire struct {
	ID          int        `json:"id"`
	Created     time.Time  `json:"created"`
	Expiration  *time.Time `json:"expiration"`
	Slug        string     `json:"slug"`
	Document    int        `json:"document"`
	FileVersion string     `json:"file_version"`
}

// shareLinkCreateWire mirrors the POST body: the server generates the slug
// and ID, and an omitted file version means the server default (archive).
type shareLinkCreateWire struct {
	Document    int        `json:"document"`
	FileVersion string     `json:"file_version"`
	Expiration  *time.Time `json:"expiration"`
}

// routeShareLinks serves the share-links family. It reports whether the
// request matched.
func (s *Server) routeShareLinks(w http.ResponseWriter, r *http.Request) bool {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/share_links/"), "/")

	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleShareLinkList(w, r)

			return true
		case http.MethodPost:
			s.handleShareLinkCreate(w, r)

			return true
		default:
			s.methodNotAllowed(w, r, http.MethodGet, http.MethodPost)

			return true
		}
	}

	id, err := strconv.Atoi(rest)
	if err != nil {
		return false
	}

	if r.Method != http.MethodDelete {
		s.methodNotAllowed(w, r, http.MethodDelete)

		return true
	}

	s.handleShareLinkDelete(w, id)

	return true
}

// handleShareLinkList serves the paginated share-link list.
func (s *Server) handleShareLinkList(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writePage(w, r, len(s.shareLinks), func(start, end int) any {
		wires := make([]shareLinkWire, 0, end-start)
		for _, link := range s.shareLinks[start:end] {
			wires = append(wires, shareLinkEntityWire(link))
		}

		return wires
	})
}

// handleShareLinkCreate stores one share link with a server-generated slug.
func (s *Server) handleShareLinkCreate(w http.ResponseWriter, r *http.Request) {
	var payload shareLinkCreateWire
	if err := json.Unmarshal(readBody(r), &payload); err != nil {
		s.t.Errorf("paperlesstest: decode share link create body: %v", err)
		s.writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{detailKey: invalidCreateDetail},
		)

		return
	}

	if payload.Document <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{detailKey: "document is required"})

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextShareLinkID++

	fileVersion := payload.FileVersion
	if fileVersion == "" {
		fileVersion = "archive"
	}

	link := shareLinkEntity{
		ID:          s.nextShareLinkID,
		Created:     time.Now().UTC(),
		Expiration:  payload.Expiration,
		Slug:        fmt.Sprintf("test-slug-%d", s.nextShareLinkID),
		Document:    payload.Document,
		FileVersion: fileVersion,
	}

	s.shareLinks = append(s.shareLinks, link)

	s.writeJSON(w, http.StatusCreated, shareLinkEntityWire(link))
}

// handleShareLinkDelete revokes one share link.
func (s *Server) handleShareLinkDelete(w http.ResponseWriter, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.shareLinks)

	s.shareLinks = slices.DeleteFunc(s.shareLinks, func(link shareLinkEntity) bool {
		return link.ID == id
	})

	if len(s.shareLinks) == before {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// shareLinkEntityWire converts the stored link to its wire shape.
func shareLinkEntityWire(link shareLinkEntity) shareLinkWire {
	return shareLinkWire{
		ID:          link.ID,
		Created:     link.Created,
		Expiration:  link.Expiration,
		Slug:        link.Slug,
		Document:    link.Document,
		FileVersion: link.FileVersion,
	}
}
