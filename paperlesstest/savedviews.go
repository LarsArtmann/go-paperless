package paperlesstest

import (
	"encoding/json/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// savedViewEntity is one saved view stored on the fake.
type savedViewEntity struct {
	ID              int
	Name            string
	ShowOnDashboard bool
	ShowInSidebar   bool
	SortField       string
	SortReverse     bool
	FilterRules     []savedViewRuleEntity
}

// savedViewRuleEntity is one filter rule of a stored saved view.
type savedViewRuleEntity struct {
	RuleType int
	Value    string
}

// savedViewWire mirrors the SavedView serializer's stable fields.
type savedViewWire struct {
	ID              int                 `json:"id"`
	Name            string              `json:"name"`
	ShowOnDashboard bool                `json:"show_on_dashboard"`
	ShowInSidebar   bool                `json:"show_in_sidebar"`
	SortField       string              `json:"sort_field"`
	SortReverse     bool                `json:"sort_reverse"`
	FilterRules     []savedViewRuleWire `json:"filter_rules"`
}

// savedViewRuleWire mirrors the SavedViewFilterRule serializer.
type savedViewRuleWire struct {
	RuleType int    `json:"rule_type"`
	Value    string `json:"value"`
}

// savedViewCreateWire is the POST body of a saved-view create (the ID is
// server-assigned).
type savedViewCreateWire = savedViewWire

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

	s.handleSavedViewDelete(w, id)

	return true
}

// handleSavedViewList serves the paginated saved-view list.
func (s *Server) handleSavedViewList(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writePage(w, r, len(s.savedViews), func(start, end int) any {
		wires := make([]savedViewWire, 0, end-start)
		for _, view := range s.savedViews[start:end] {
			wires = append(wires, savedViewEntityWire(view))
		}

		return wires
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

	s.nextSavedViewID++
	payload.ID = s.nextSavedViewID

	entity := savedViewEntity{
		ID:              payload.ID,
		Name:            payload.Name,
		ShowOnDashboard: payload.ShowOnDashboard,
		ShowInSidebar:   payload.ShowInSidebar,
		SortField:       payload.SortField,
		SortReverse:     payload.SortReverse,
	}

	for _, rule := range payload.FilterRules {
		entity.FilterRules = append(entity.FilterRules, savedViewRuleEntity{
			RuleType: rule.RuleType,
			Value:    rule.Value,
		})
	}

	s.savedViews = append(s.savedViews, entity)

	s.writeJSON(w, http.StatusCreated, payload)
}

// handleSavedViewDelete removes one saved view.
func (s *Server) handleSavedViewDelete(w http.ResponseWriter, id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.savedViews)

	s.savedViews = slices.DeleteFunc(s.savedViews, func(view savedViewEntity) bool {
		return view.ID == id
	})

	if len(s.savedViews) == before {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// savedViewEntityWire converts the stored view to its wire shape.
func savedViewEntityWire(view savedViewEntity) savedViewWire {
	wire := savedViewWire{
		ID:              view.ID,
		Name:            view.Name,
		ShowOnDashboard: view.ShowOnDashboard,
		ShowInSidebar:   view.ShowInSidebar,
		SortField:       view.SortField,
		SortReverse:     view.SortReverse,
		FilterRules:     []savedViewRuleWire{},
	}

	for _, rule := range view.FilterRules {
		wire.FilterRules = append(wire.FilterRules, savedViewRuleWire{
			RuleType: rule.RuleType,
			Value:    rule.Value,
		})
	}

	return wire
}
