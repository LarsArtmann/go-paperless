package paperlesstest

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Route prefixes of the named-entity families the fake serves.
const (
	prefixTags           = "/api/tags/"
	prefixCorrespondents = "/api/correspondents/"
	prefixDocumentTypes  = "/api/document_types/"
	prefixCustomFields   = "/api/custom_fields/"
	prefixStoragePaths   = "/api/storage_paths/"
)

// namedEntity is one tag, correspondent, or document type on the fake.
type namedEntity struct {
	ID                int
	Name              string
	MatchingAlgorithm int
}

// namedEntityWire mirrors the create/list subset shared by Paperless-ngx's
// tag and correspondent objects (id + name + matching algorithm).
type namedEntityWire struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	MatchingAlgorithm int    `json:"matching_algorithm"`
}

// customFieldEntity is one custom-field definition on the fake.
type customFieldEntity struct {
	ID       int
	Name     string
	DataType string
}

// customFieldWire mirrors one custom field definition.
type customFieldWire struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

// storagePathEntity is one storage path definition on the fake.
type storagePathEntity struct {
	ID   int
	Slug string
	Name string
	Path string
}

// storagePathWire mirrors Paperless-ngx's storage path serializer fields
// the SDK relies on.
type storagePathWire struct {
	ID   int    `json:"id"`
	Slug string `json:"slug,omitempty"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// matchingAlgorithmWire is the PATCH body for a named object's matching
// algorithm.
type matchingAlgorithmWire struct {
	MatchingAlgorithm int `json:"matching_algorithm"`
}

// WithTags seeds tags (matching algorithm 0 = none, the deterministic
// provenance default).
func WithTags(names ...string) Option {
	return func(s *Server) {
		for _, name := range names {
			s.AddTag(name, 0)
		}
	}
}

// matchingAlgorithmAuto is Paperless-ngx's "auto" matching algorithm, the
// default the fake seeds correspondents with (mirroring the SDK's
// EnsureCorrespondent creation policy).
const matchingAlgorithmAuto = 6

// WithCorrespondents seeds correspondents (auto matching, mirroring the
// SDK's EnsureCorrespondent default).
func WithCorrespondents(names ...string) Option {
	return func(s *Server) {
		for _, name := range names {
			s.AddCorrespondent(name, matchingAlgorithmAuto)
		}
	}
}

// WithDocumentTypes seeds document types (matching algorithm 0 = none).
func WithDocumentTypes(names ...string) Option {
	return func(s *Server) {
		for _, name := range names {
			s.AddDocumentType(name, 0)
		}
	}
}

// WithCustomFields seeds custom-field definitions as string fields.
func WithCustomFields(names ...string) Option {
	return func(s *Server) {
		for _, name := range names {
			s.AddCustomField(name, "string")
		}
	}
}

// StoragePathFixture is one storage path to seed via WithStoragePaths.
type StoragePathFixture struct {
	Name string
	Path string
}

// WithStoragePaths seeds storage path definitions.
func WithStoragePaths(paths ...StoragePathFixture) Option {
	return func(s *Server) {
		for _, path := range paths {
			s.AddStoragePath(path.Name, path.Path)
		}
	}
}

// AddTag stores one tag and returns its ID (the runtime seeding entry
// point; use it mid-test, e.g. to plant a legacy "auto" tag for the
// self-heal flow).
func (s *Server) AddTag(name string, matchingAlgorithm int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextTagID++

	entity := namedEntity{ID: s.nextTagID, Name: name, MatchingAlgorithm: matchingAlgorithm}
	s.tags = append(s.tags, entity)

	return entity.ID
}

// AddCorrespondent stores one correspondent and returns its ID.
func (s *Server) AddCorrespondent(name string, matchingAlgorithm int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextCorrespondentID++

	entity := namedEntity{
		ID:                s.nextCorrespondentID,
		Name:              name,
		MatchingAlgorithm: matchingAlgorithm,
	}
	s.correspondents = append(s.correspondents, entity)

	return entity.ID
}

// AddDocumentType stores one document type and returns its ID.
func (s *Server) AddDocumentType(name string, matchingAlgorithm int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextDocumentTypeID++

	entity := namedEntity{
		ID:                s.nextDocumentTypeID,
		Name:              name,
		MatchingAlgorithm: matchingAlgorithm,
	}
	s.documentTypes = append(s.documentTypes, entity)

	return entity.ID
}

// AddCustomField stores one custom-field definition and returns its ID.
func (s *Server) AddCustomField(name, dataType string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextCustomFieldID++

	entity := customFieldEntity{ID: s.nextCustomFieldID, Name: name, DataType: dataType}
	s.customFields = append(s.customFields, entity)

	return entity.ID
}

// AddStoragePath stores one storage path definition and returns its ID.
func (s *Server) AddStoragePath(name, path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextStoragePathID++

	entity := storagePathEntity{
		ID:   s.nextStoragePathID,
		Slug: fmt.Sprintf("path-%d", s.nextStoragePathID),
		Name: name,
		Path: path,
	}
	s.storagePaths = append(s.storagePaths, entity)

	return entity.ID
}

// routeEntities serves the five named-entity families. It reports whether
// the request matched one of their routes.
func (s *Server) routeEntities(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case strings.HasPrefix(r.URL.Path, prefixTags):
		return s.routeNamedFamily(w, r, prefixTags, &s.tags, &s.nextTagID)
	case strings.HasPrefix(r.URL.Path, prefixCorrespondents):
		return s.routeNamedFamily(
			w,
			r,
			prefixCorrespondents,
			&s.correspondents,
			&s.nextCorrespondentID,
		)
	case strings.HasPrefix(r.URL.Path, prefixDocumentTypes):
		return s.routeNamedFamily(
			w,
			r,
			prefixDocumentTypes,
			&s.documentTypes,
			&s.nextDocumentTypeID,
		)
	case strings.HasPrefix(r.URL.Path, prefixCustomFields):
		return s.routeCustomFields(w, r)
	case strings.HasPrefix(r.URL.Path, prefixStoragePaths):
		return s.routeStoragePaths(w, r)
	default:
		return false
	}
}

// routeNamedFamily serves one tag/correspondent/document-type family:
// a filtered list, creates, detail lookups, and the matching-algorithm
// PATCH the tag self-heal flow uses. It reports whether the request
// matched.
func (s *Server) routeNamedFamily(
	w http.ResponseWriter,
	r *http.Request,
	prefix string,
	store *[]namedEntity,
	nextID *int,
) bool {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")

	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleNamedList(w, r, store)

			return true
		case http.MethodPost:
			s.handleNamedCreate(w, r, store, nextID)

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

	switch r.Method {
	case http.MethodGet:
		s.handleNamedDetail(w, id, store)

		return true
	case http.MethodPatch:
		s.handleNamedPatch(w, r, id, store)

		return true
	default:
		s.methodNotAllowed(w, r, http.MethodGet, http.MethodPatch)

		return true
	}
}

// handleNamedList serves the family's paginated list, honoring the
// name__iexact filter the SDK's by-name lookups send.
func (s *Server) handleNamedList(w http.ResponseWriter, r *http.Request, store *[]namedEntity) {
	filter := r.URL.Query().Get("name__iexact")

	s.mu.Lock()
	defer s.mu.Unlock()

	matches := make([]namedEntity, 0, len(*store))
	for _, entity := range *store {
		if filter == "" || strings.EqualFold(entity.Name, filter) {
			matches = append(matches, entity)
		}
	}

	s.writePage(w, r, len(matches), func(start, end int) any {
		wires := make([]namedEntityWire, 0, end-start)
		for _, entity := range matches[start:end] {
			wires = append(wires, namedEntityWire(entity))
		}

		return wires
	})
}

// handleNamedCreate stores one named object and answers the created
// payload.
func (s *Server) handleNamedCreate(
	w http.ResponseWriter,
	r *http.Request,
	store *[]namedEntity,
	nextID *int,
) {
	var payload namedEntityWire
	if err := json.Unmarshal(readBody(r), &payload); err != nil {
		s.t.Errorf("paperlesstest: decode %s create body: %v", r.URL.Path, err)
		s.writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{detailKey: invalidCreateDetail},
		)

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	*nextID++
	payload.ID = *nextID

	*store = append(*store, namedEntity(payload))

	s.writeJSON(w, http.StatusCreated, payload)
}

// handleNamedDetail serves one named object's payload by ID.
func (s *Server) handleNamedDetail(w http.ResponseWriter, id int, store *[]namedEntity) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entity := findNamedLocked(*store, id)
	if entity == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	s.writeJSON(w, http.StatusOK, namedEntityWire(*entity))
}

// handleNamedPatch applies a matching-algorithm patch (the self-heal
// flow's write).
func (s *Server) handleNamedPatch(
	w http.ResponseWriter,
	r *http.Request,
	id int,
	store *[]namedEntity,
) {
	var patch matchingAlgorithmWire
	if err := json.Unmarshal(readBody(r), &patch); err != nil {
		s.t.Errorf("paperlesstest: decode %s patch body: %v", r.URL.Path, err)
		s.writeJSON(w, http.StatusBadRequest, map[string]string{detailKey: "invalid patch payload"})

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entity := findNamedLocked(*store, id)
	if entity == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	entity.MatchingAlgorithm = patch.MatchingAlgorithm

	s.writeJSON(w, http.StatusOK, namedEntityWire(*entity))
}

// routeCustomFields serves the custom-field definitions family.
func (s *Server) routeCustomFields(w http.ResponseWriter, r *http.Request) bool {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefixCustomFields), "/")

	if rest != "" {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		filter := r.URL.Query().Get("name__iexact")

		s.mu.Lock()
		defer s.mu.Unlock()

		matches := make([]customFieldEntity, 0, len(s.customFields))
		for _, field := range s.customFields {
			if filter == "" || strings.EqualFold(field.Name, filter) {
				matches = append(matches, field)
			}
		}

		s.writePage(w, r, len(matches), func(start, end int) any {
			wires := make([]customFieldWire, 0, end-start)
			for _, field := range matches[start:end] {
				wires = append(wires, customFieldWire(field))
			}

			return wires
		})

		return true
	case http.MethodPost:
		var payload customFieldWire
		if err := json.Unmarshal(readBody(r), &payload); err != nil {
			s.t.Errorf("paperlesstest: decode custom field create body: %v", err)
			s.writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{detailKey: invalidCreateDetail},
			)

			return true
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		s.nextCustomFieldID++
		payload.ID = s.nextCustomFieldID

		if payload.DataType == "" {
			payload.DataType = "string"
		}

		s.customFields = append(s.customFields, customFieldEntity(payload))

		s.writeJSON(w, http.StatusCreated, payload)

		return true
	default:
		s.methodNotAllowed(w, r, http.MethodGet, http.MethodPost)

		return true
	}
}

// routeStoragePaths serves the storage path definitions family.
func (s *Server) routeStoragePaths(w http.ResponseWriter, r *http.Request) bool {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefixStoragePaths), "/")

	if rest != "" {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		filter := r.URL.Query().Get("name__iexact")

		s.mu.Lock()
		defer s.mu.Unlock()

		matches := make([]storagePathEntity, 0, len(s.storagePaths))
		for _, path := range s.storagePaths {
			if filter == "" || strings.EqualFold(path.Name, filter) {
				matches = append(matches, path)
			}
		}

		s.writePage(w, r, len(matches), func(start, end int) any {
			wires := make([]storagePathWire, 0, end-start)
			for _, path := range matches[start:end] {
				wires = append(wires, storagePathWire(path))
			}

			return wires
		})

		return true
	case http.MethodPost:
		var payload storagePathWire
		if err := json.Unmarshal(readBody(r), &payload); err != nil {
			s.t.Errorf("paperlesstest: decode storage path create body: %v", err)
			s.writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{detailKey: invalidCreateDetail},
			)

			return true
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		s.nextStoragePathID++
		payload.ID = s.nextStoragePathID
		payload.Slug = fmt.Sprintf("path-%d", payload.ID)

		s.storagePaths = append(s.storagePaths, storagePathEntity(payload))

		s.writeJSON(w, http.StatusCreated, payload)

		return true
	default:
		s.methodNotAllowed(w, r, http.MethodGet, http.MethodPost)

		return true
	}
}

// findNamedLocked locates one named entity by ID, or nil. The caller must
// hold s.mu.
func findNamedLocked(store []namedEntity, id int) *namedEntity {
	for index := range store {
		if store[index].ID == id {
			return &store[index]
		}
	}

	return nil
}
