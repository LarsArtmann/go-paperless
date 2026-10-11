package paperlesstest

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	id "github.com/larsartmann/go-branded-id"
	"github.com/larsartmann/go-paperless"
)

// Route prefixes of the named-entity families the fake serves.
const (
	prefixTags           = "/api/tags/"
	prefixCorrespondents = "/api/correspondents/"
	prefixDocumentTypes  = "/api/document_types/"
	prefixCustomFields   = "/api/custom_fields/"
	prefixStoragePaths   = "/api/storage_paths/"
)

// namedEntity is one tag, correspondent, or document type. The fake stores
// and serves the same shape, so the struct carries the wire tags directly;
// the brand parameter binds the stored ID to the entity family's branded
// type.
type namedEntity[B any] struct { // SDK↔fake wire-shape mirror: the SDK's payload is unexported, so the fake re-declares it; the round-trip contract depends on it
	ID                id.ID[B, int] `json:"id"`
	Name              string        `json:"name"`
	MatchingAlgorithm int           `json:"matching_algorithm"`
}

// customField is one custom-field definition. The fake stores and serves
// the same shape, so the struct carries the wire tags directly.
type customField struct {
	ID       paperless.CustomFieldID `json:"id"`
	Name     string                  `json:"name"`
	DataType string                  `json:"data_type"`
}

// storagePath is one storage path definition. The fake stores and serves
// the same shape, so the struct carries the wire tags the SDK relies on
// directly.
type storagePath struct {
	ID   paperless.StoragePathID `json:"id"`
	Slug string                  `json:"slug,omitempty"`
	Name string                  `json:"name"`
	Path string                  `json:"path"`
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
func (s *Server) AddTag(name string, matchingAlgorithm int) paperless.TagID {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextTagID = paperless.NewTagID(s.nextTagID.Get() + 1)

	entity := namedEntity[paperless.TagBrand]{
		ID:                s.nextTagID,
		Name:              name,
		MatchingAlgorithm: matchingAlgorithm,
	}
	s.tags = append(s.tags, entity)

	return entity.ID
}

// AddCorrespondent stores one correspondent and returns its ID.
func (s *Server) AddCorrespondent(name string, matchingAlgorithm int) paperless.CorrespondentID {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextCorrespondentID = paperless.NewCorrespondentID(s.nextCorrespondentID.Get() + 1)

	entity := namedEntity[paperless.CorrespondentBrand]{
		ID:                s.nextCorrespondentID,
		Name:              name,
		MatchingAlgorithm: matchingAlgorithm,
	}
	s.correspondents = append(s.correspondents, entity)

	return entity.ID
}

// AddDocumentType stores one document type and returns its ID.
func (s *Server) AddDocumentType(name string, matchingAlgorithm int) paperless.DocumentTypeID {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextDocumentTypeID = paperless.NewDocumentTypeID(s.nextDocumentTypeID.Get() + 1)

	entity := namedEntity[paperless.DocumentTypeBrand]{
		ID:                s.nextDocumentTypeID,
		Name:              name,
		MatchingAlgorithm: matchingAlgorithm,
	}
	s.documentTypes = append(s.documentTypes, entity)

	return entity.ID
}

// AddCustomField stores one custom-field definition and returns its ID.
func (s *Server) AddCustomField(
	name, dataType string,
) paperless.CustomFieldID { //nolint:branching-flow // distinct concepts (label vs serializer type tag); branded params are the ROADMAP v2 typed-API scope
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextCustomFieldID = paperless.NewCustomFieldID(s.nextCustomFieldID.Get() + 1)

	entity := customField{ID: s.nextCustomFieldID, Name: name, DataType: dataType}
	s.customFields = append(s.customFields, entity)

	return entity.ID
}

// AddStoragePath stores one storage path definition and returns its ID.
func (s *Server) AddStoragePath(
	name, path string,
) paperless.StoragePathID { //nolint:branching-flow // distinct concepts (label vs directory template); branded params are the ROADMAP v2 typed-API scope
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextStoragePathID = paperless.NewStoragePathID(s.nextStoragePathID.Get() + 1)

	entity := storagePath{
		ID:   s.nextStoragePathID,
		Slug: fmt.Sprintf("path-%d", s.nextStoragePathID.Get()),
		Name: name,
		Path: path,
	}
	s.storagePaths = append(s.storagePaths, entity)

	return entity.ID
}

// CorrespondentCount reports how many correspondents exist on the fake
// (dry-run purity assertions).
func (s *Server) CorrespondentCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.correspondents)
}

// DocumentTypeID looks one seeded or created document type up by exact
// name and reports whether it exists.
func (s *Server) DocumentTypeID(name string) (paperless.DocumentTypeID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entity := range s.documentTypes {
		if entity.Name == name {
			return entity.ID, true
		}
	}

	return paperless.DocumentTypeID{}, false
}

// CustomFieldID looks one seeded or created custom-field definition up by
// exact name and reports whether it exists.
func (s *Server) CustomFieldID(name string) (paperless.CustomFieldID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, field := range s.customFields {
		if field.Name == name {
			return field.ID, true
		}
	}
	return paperless.CustomFieldID{}, false
}

// routeEntities serves the five named-entity families. It reports whether
// the request matched one of their routes.
func (s *Server) routeEntities(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case strings.HasPrefix(r.URL.Path, prefixTags):
		return routeNamedFamily(s, w, r, prefixTags, &s.tags, &s.nextTagID)
	case strings.HasPrefix(r.URL.Path, prefixCorrespondents):
		return routeNamedFamily(
			s,
			w,
			r,
			prefixCorrespondents,
			&s.correspondents,
			&s.nextCorrespondentID,
		)
	case strings.HasPrefix(r.URL.Path, prefixDocumentTypes):
		return routeNamedFamily(
			s,
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
// matched. It is a free generic function because Go methods cannot carry
// type parameters; the brand binds the family's store and IDs to one
// entity type.
func routeNamedFamily[B any](
	server *Server,
	w http.ResponseWriter,
	r *http.Request,
	prefix string,
	store *[]namedEntity[B],
	nextID *id.ID[B, int],
) bool {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")

	if rest == "" {
		switch r.Method {
		case http.MethodGet:
			handleNamedList(server, w, r, store)

			return true
		case http.MethodPost:
			handleNamedCreate(server, w, r, store, nextID)

			return true
		default:
			server.methodNotAllowed(w, r, http.MethodGet, http.MethodPost)

			return true
		}
	}

	rawID, err := strconv.Atoi(rest)
	if err != nil {
		return false
	}

	entityID := id.NewID[B](rawID)

	switch r.Method {
	case http.MethodGet:
		handleNamedDetail(server, w, entityID, store)

		return true
	case http.MethodPatch:
		handleNamedPatch(server, w, r, entityID, store)

		return true
	default:
		server.methodNotAllowed(w, r, http.MethodGet, http.MethodPatch)

		return true
	}
}

// handleNamedList serves the family's paginated list, honoring the
// name__iexact filter the SDK's by-name lookups send.
func handleNamedList[B any](
	server *Server,
	w http.ResponseWriter,
	r *http.Request,
	store *[]namedEntity[B],
) {
	filter := r.URL.Query().Get("name__iexact")

	server.mu.Lock()
	defer server.mu.Unlock()

	matches := make([]namedEntity[B], 0, len(*store))
	for _, entity := range *store {
		if filter == "" || strings.EqualFold(entity.Name, filter) {
			matches = append(matches, entity)
		}
	}

	server.writePage(w, r, len(matches), func(start, end int) any {
		entities := make([]namedEntity[B], 0, end-start)
		entities = append(entities, matches[start:end]...)

		return entities
	})
}

// handleNamedCreate stores one named object and answers the created
// payload.
func handleNamedCreate[B any](
	server *Server,
	w http.ResponseWriter,
	r *http.Request,
	store *[]namedEntity[B],
	nextID *id.ID[B, int],
) {
	var payload namedEntity[B]
	if err := json.Unmarshal(readBody(r), &payload); err != nil {
		server.t.Errorf("paperlesstest: decode %s create body: %v", r.URL.Path, err)
		server.writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{detailKey: invalidCreateDetail},
		)

		return
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	*nextID = id.NewID[B](nextID.Get() + 1)
	payload.ID = *nextID

	*store = append(*store, payload)

	server.writeJSON(w, http.StatusCreated, payload)
}

// handleNamedDetail serves one named object's payload by ID.
func handleNamedDetail[B any](
	server *Server,
	w http.ResponseWriter,
	entityID id.ID[B, int],
	store *[]namedEntity[B],
) {
	server.mu.Lock()
	defer server.mu.Unlock()

	entity := findNamedLocked(*store, entityID)
	if entity == nil {
		server.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	server.writeJSON(w, http.StatusOK, *entity)
}

// handleNamedPatch applies a matching-algorithm patch (the self-heal
// flow's write).
func handleNamedPatch[B any](
	server *Server,
	w http.ResponseWriter,
	r *http.Request,
	entityID id.ID[B, int],
	store *[]namedEntity[B],
) {
	var patch matchingAlgorithmWire
	if err := json.Unmarshal(readBody(r), &patch); err != nil {
		server.t.Errorf("paperlesstest: decode %s patch body: %v", r.URL.Path, err)
		server.writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{detailKey: "invalid patch payload"},
		)

		return
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	entity := findNamedLocked(*store, entityID)
	if entity == nil {
		server.writeJSON(w, http.StatusNotFound, map[string]string{detailKey: notFoundDetail})

		return
	}

	entity.MatchingAlgorithm = patch.MatchingAlgorithm

	server.writeJSON(w, http.StatusOK, *entity)
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

		matches := make([]customField, 0, len(s.customFields))
		for _, field := range s.customFields {
			if filter == "" || strings.EqualFold(field.Name, filter) {
				matches = append(matches, field)
			}
		}

		s.writePage(w, r, len(matches), func(start, end int) any {
			fields := make([]customField, 0, end-start)
			fields = append(fields, matches[start:end]...)

			return fields
		})

		return true
	case http.MethodPost:
		var payload customField
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

		s.nextCustomFieldID = paperless.NewCustomFieldID(s.nextCustomFieldID.Get() + 1)
		payload.ID = s.nextCustomFieldID

		if payload.DataType == "" {
			payload.DataType = "string"
		}

		s.customFields = append(s.customFields, payload)

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

		matches := make([]storagePath, 0, len(s.storagePaths))
		for _, path := range s.storagePaths {
			if filter == "" || strings.EqualFold(path.Name, filter) {
				matches = append(matches, path)
			}
		}

		s.writePage(w, r, len(matches), func(start, end int) any {
			paths := make([]storagePath, 0, end-start)
			paths = append(paths, matches[start:end]...)

			return paths
		})

		return true
	case http.MethodPost:
		var payload storagePath
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

		s.nextStoragePathID = paperless.NewStoragePathID(s.nextStoragePathID.Get() + 1)
		payload.ID = s.nextStoragePathID
		payload.Slug = fmt.Sprintf("path-%d", payload.ID.Get())

		s.storagePaths = append(s.storagePaths, payload)

		s.writeJSON(w, http.StatusCreated, payload)

		return true
	default:
		s.methodNotAllowed(w, r, http.MethodGet, http.MethodPost)

		return true
	}
}

// findNamedLocked locates one named entity by ID, or nil. The caller must
// hold s.mu.
func findNamedLocked[B any](store []namedEntity[B], entityID id.ID[B, int]) *namedEntity[B] {
	for index := range store {
		if store[index].ID.Equal(entityID) {
			return &store[index]
		}
	}

	return nil
}
