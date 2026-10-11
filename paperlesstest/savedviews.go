package paperlesstest

import (
	"encoding/json/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/larsartmann/go-paperless"
)

// savedViewRule is one filter rule of a saved view. The fake stores and
// serves the same shape, so the struct carries the SavedViewFilterRule
// serializer's wire tags directly.
//
// SDK↔fake wire-shape mirror: the SDK's payload is unexported, so the fake re-declares it; the round-trip
// contract depends on it.
type savedViewRule struct {
	RuleType int    `json:"rule_type"`
	Value    string `json:"value"`
}

// savedView is one saved view. The fake stores and serves the same shape,
// so the struct carries the SavedView serializer's stable-field wire tags
// directly.
type savedView struct {
	ID              paperless.SavedViewID `json:"id"`
	Name            string                `json:"name"`
	ShowOnDashboard bool                  `json:"show_on_dashboard"`
	ShowInSidebar   bool                  `json:"show_in_sidebar"`
	SortField       string                `json:"sort_field"`
	SortReverse     bool                  `json:"sort_reverse"`
	FilterRules     []savedViewRule       `json:"filter_rules"`
}

// savedViewCreateWire is the POST body of a saved-view create (the ID is
// server-assigned).
type savedViewCreateWire = savedView

// routeSavedViews serves the saved-views family. It reports whether the
// request matched.
func (s *Server) routeSavedViews(w http.ResponseWriter, r *http.Request) bool {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/saved_views/"), "/")

	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleSavedViewList(w, r)

			return true
		case http.MethodPost:
			s.handleSavedViewCreate(w, r)

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

	s.handleSavedViewDelete(w, paperless.NewSavedViewID(id))

	return true
}

// handleSavedViewList serves the paginated saved-view list.
func (s *Server) handleSavedViewList(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writePage(w, r, len(s.savedViews), func(start, end int) any {
		views := make([]savedView, 0, end-start)
		views = append(views, s.savedViews[start:end]...)

		return views
	})
}

// handleSavedViewCreate stores one saved view and answers its payload.
func (s *Server) handleSavedViewCreate(w http.ResponseWriter, r *http.Request) {
	var payload savedViewCreateWire
	if err := json.Unmarshal(readBody(r), &payload); err != nil {
		s.t.Errorf("paperlesstest: decode saved view create body: %v", err)
		s.writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{detailKey: invalidCreateDetail},
		)

		return
	}

	if payload.Name == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{detailKey: "name is required"})

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSavedViewID = paperless.NewSavedViewID(s.nextSavedViewID.Get() + 1)
	payload.ID = s.nextSavedViewID

	if payload.FilterRules == nil {
		payload.FilterRules = []savedViewRule{}
	}

	s.savedViews = append(s.savedViews, payload)

	s.writeJSON(w, http.StatusCreated, payload)
}

// handleSavedViewDelete removes one saved view.
func (s *Server) handleSavedViewDelete(w http.ResponseWriter, viewID paperless.SavedViewID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.savedViews)

	s.savedViews = slices.DeleteFunc(s.savedViews, func(view savedView) bool {
		return view.ID.Equal(viewID)
	})

	if len(s.savedViews) == before {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
