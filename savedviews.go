package paperless

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"
)

// SavedViewRuleType is Paperless-ngx's saved-view filter rule_type enum
// (src-ui/src/app/data/filter-rule-type.ts, mirrored server-side in
// documents/models.py). A distinct type keeps saved-view rules from
// carrying arbitrary ints; the constants below are the common filters,
// not the full upstream table — servers add IDs over time, so unknown
// values decode and re-encode unchanged.
type SavedViewRuleType int

const (
	// SavedViewRuleTypeTitle matches the title text (upstream
	// title__icontains; deprecated upstream in favor of title_search but
	// still present in existing saved views).
	SavedViewRuleTypeTitle SavedViewRuleType = 0
	// SavedViewRuleTypeContent matches the document's OCR content.
	SavedViewRuleTypeContent SavedViewRuleType = 1
	// SavedViewRuleTypeASN matches the archive serial number.
	SavedViewRuleTypeASN SavedViewRuleType = 2
	// SavedViewRuleTypeCorrespondent matches a correspondent by ID.
	SavedViewRuleTypeCorrespondent SavedViewRuleType = 3
	// SavedViewRuleTypeDocumentType matches a document type by ID.
	SavedViewRuleTypeDocumentType SavedViewRuleType = 4
	// SavedViewRuleTypeIsInInbox matches documents in the inbox.
	SavedViewRuleTypeIsInInbox SavedViewRuleType = 5
	// SavedViewRuleTypeHasTagsAll requires every listed tag.
	SavedViewRuleTypeHasTagsAll SavedViewRuleType = 6
	// SavedViewRuleTypeHasAnyTag matches documents carrying at least one
	// tag (upstream is_tagged).
	SavedViewRuleTypeHasAnyTag SavedViewRuleType = 7
	// SavedViewRuleTypeCreatedBefore matches created dates before a value.
	SavedViewRuleTypeCreatedBefore SavedViewRuleType = 8
	// SavedViewRuleTypeCreatedAfter matches created dates after a value.
	SavedViewRuleTypeCreatedAfter SavedViewRuleType = 9
	// SavedViewRuleTypeAddedBefore matches added dates before a value.
	SavedViewRuleTypeAddedBefore SavedViewRuleType = 13
	// SavedViewRuleTypeAddedAfter matches added dates after a value.
	SavedViewRuleTypeAddedAfter SavedViewRuleType = 14
	// SavedViewRuleTypeDoesNotHaveTag excludes documents carrying the tag.
	SavedViewRuleTypeDoesNotHaveTag SavedViewRuleType = 17
	// SavedViewRuleTypeFullTextQuery runs the upstream full-text query
	// syntax (upstream query filtervar).
	SavedViewRuleTypeFullTextQuery SavedViewRuleType = 20
	// SavedViewRuleTypeHasTagsAny requires at least one of the listed tags.
	SavedViewRuleTypeHasTagsAny SavedViewRuleType = 22
	// SavedViewRuleTypeStoragePath matches a storage path by ID.
	SavedViewRuleTypeStoragePath SavedViewRuleType = 25
)

// SavedViewFilterRule is one filter criterion of a saved view: the
// server's numeric rule type (e.g. 6 = require all of the listed tags)
// and its string value.
type SavedViewFilterRule struct {
	RuleType SavedViewRuleType
	Value    string
}

// savedViewFilterRulePayload mirrors the SavedViewFilterRule serializer.
//
// SDK↔fake wire-shape mirror is deliberate: the fake must serialize exactly the SDK's parse shape.
type savedViewFilterRulePayload struct {
	RuleType int    `json:"rule_type"`
	Value    string `json:"value"`
}

// SavedView is one stored filter view ("Inbox", ...) in the web UI. The
// SDK models the stable core fields; servers add or drop UI fields
// (icon, page_size, display mode, ...) across versions and unknown JSON
// fields are ignored.
//
// Embedding the shared fields would change the public struct's composite-literal surface; unify them in the
// ROADMAP v2 typed-API scope instead.
type SavedView struct {
	ID   SavedViewID
	Name string
	// ShowOnDashboard and ShowInSidebar mirror the view's visibility
	// flags; a server that no longer serves them decodes as false.
	ShowOnDashboard bool
	ShowInSidebar   bool
	SortField       string
	SortReverse     bool
	FilterRules     []SavedViewFilterRule
}

// savedViewPayload mirrors the SavedView serializer's stable fields.
//
// Unexported wire mirror; it deliberately differs from the create request (server-assigned fields).
type savedViewPayload struct {
	ID              SavedViewID                  `json:"id"`
	Name            string                       `json:"name"`
	ShowOnDashboard bool                         `json:"show_on_dashboard"`
	ShowInSidebar   bool                         `json:"show_in_sidebar"`
	SortField       string                       `json:"sort_field"`
	SortReverse     bool                         `json:"sort_reverse"`
	FilterRules     []savedViewFilterRulePayload `json:"filter_rules"`
}

func (p savedViewPayload) savedView() SavedView {
	view := SavedView{
		ID:              p.ID,
		Name:            p.Name,
		ShowOnDashboard: p.ShowOnDashboard,
		ShowInSidebar:   p.ShowInSidebar,
		SortField:       p.SortField,
		SortReverse:     p.SortReverse,
		FilterRules:     make([]SavedViewFilterRule, 0, len(p.FilterRules)),
	}

	for _, rule := range p.FilterRules {
		view.FilterRules = append(view.FilterRules, SavedViewFilterRule{
			RuleType: SavedViewRuleType(rule.RuleType),
			Value:    rule.Value,
		})
	}

	return view
}

// CreateSavedViewRequest asks for one saved view. Name is required; the
// filter rules carry the server's numeric rule types.
//
// Embedding the shared fields would change the public struct's composite-literal surface; unify them in the
// ROADMAP v2 typed-API scope instead.
type CreateSavedViewRequest struct {
	Name            string
	ShowOnDashboard bool
	ShowInSidebar   bool
	SortField       string
	SortReverse     bool
	FilterRules     []SavedViewFilterRule
}

// ListSavedViews returns every saved view on the server, paginating the
// same bounded way as the document listings.
func (c *Client) ListSavedViews(ctx context.Context) ([]SavedView, error) {
	payloads, listErr := fetchAllPages[savedViewPayload](ctx, c, pathSavedViews, "saved view")
	if listErr != nil {
		return nil, listErr
	}

	views := make([]SavedView, 0, len(payloads))

	for _, payload := range payloads {
		views = append(views, payload.savedView())
	}

	return views, nil
}

// CreateSavedView stores one saved view and returns its ID.
func (c *Client) CreateSavedView(
	ctx context.Context,
	req CreateSavedViewRequest,
) (SavedViewID, error) {
	if req.Name == "" {
		return SavedViewID{}, errorfamily.NewRejection("paperless.empty_saved_view",
			"saved view name is required")
	}

	rules := make([]savedViewFilterRulePayload, 0, len(req.FilterRules))
	for _, rule := range req.FilterRules {
		rules = append(rules, savedViewFilterRulePayload{
			RuleType: int(rule.RuleType),
			Value:    rule.Value,
		})
	}

	payload, err := json.Marshal(struct {
		Name            string                       `json:"name"`
		ShowOnDashboard bool                         `json:"show_on_dashboard"`
		ShowInSidebar   bool                         `json:"show_in_sidebar"`
		SortField       string                       `json:"sort_field"`
		SortReverse     bool                         `json:"sort_reverse"`
		FilterRules     []savedViewFilterRulePayload `json:"filter_rules"`
	}{
		Name:            req.Name,
		ShowOnDashboard: req.ShowOnDashboard,
		ShowInSidebar:   req.ShowInSidebar,
		SortField:       req.SortField,
		SortReverse:     req.SortReverse,
		FilterRules:     rules,
	})
	if err != nil {
		return SavedViewID{}, errorfamily.WrapInfrastructure(err, "paperless.marshal_saved_view",
			"could not encode the saved view payload").
			WithContext("name", req.Name)
	}

	raw, err := c.doRequest(
		ctx,
		http.MethodPost,
		pathSavedViews,
		"",
		bytes.NewReader(payload),
		"application/json",
	)
	if err != nil {
		return SavedViewID{}, fmt.Errorf(
			"create saved view %q: %w",
			req.Name,
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	var created savedViewPayload
	if err := json.Unmarshal(raw, &created); err != nil {
		return SavedViewID{}, errorfamily.WrapCorruption(err, "paperless.decode_saved_view",
			"could not decode the created saved view").
			WithContext("name", req.Name)
	}

	return created.ID, nil
}

// DeleteSavedView removes one saved view from the server.
func (c *Client) DeleteSavedView(ctx context.Context, viewID SavedViewID) error {
	if viewID.IsZero() {
		return errorfamily.NewRejection("paperless.empty_saved_view_id",
			"saved view ID is required to delete a saved view")
	}

	if _, err := c.doRequest(
		ctx,
		http.MethodDelete,
		pathSavedViews+strconv.Itoa(viewID.Get())+"/",
		"",
		nil,
		"",
	); err != nil {
		return fmt.Errorf(
			"delete saved view %d: %w",
			viewID.Get(),
			err,
		) //nolint:erraudit // keeps inner code+family
	}

	return nil
}
