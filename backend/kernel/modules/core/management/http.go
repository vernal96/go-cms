package management

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/modules/resourceextension"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type contentHTTP struct {
	sites     *Sites
	resources *Resources
}

func registerContentRoutes(router chi.Router, sites *Sites, resources *Resources) {
	handler := &contentHTTP{sites: sites, resources: resources}
	router.Get("/sites/options", handler.listSiteOptions)
	router.Get("/sites", handler.listSites)
	router.Post("/sites", handler.createSite)
	router.Get("/site-profiles", handler.listProfiles)
	router.Get("/sites/{siteID}", handler.getSite)
	router.Patch("/sites/{siteID}", handler.updateSite)
	router.Delete("/sites/{siteID}", handler.deleteSite)
	router.Get("/sites/{siteID}/resources", handler.listResourceChildren)
	router.Post("/sites/{siteID}/resources", handler.createResource)
	router.Get("/sites/{siteID}/resources/metadata", handler.resourceMetadata)
	router.Get("/sites/{siteID}/resources/options", handler.resourceOptions)
	router.Get("/sites/{siteID}/resources/lookup", handler.resourceLookup)
	router.Get("/sites/{siteID}/resources/{resourceID}", handler.getResource)
	router.Patch("/sites/{siteID}/resources/{resourceID}", handler.updateResource)
	router.Post("/sites/{siteID}/resources/{resourceID}/widgets", handler.createResourceWidget)
	router.Patch("/sites/{siteID}/resources/{resourceID}/widgets/{widgetID}", handler.updateResourceWidget)
	router.Delete("/sites/{siteID}/resources/{resourceID}/widgets/{widgetID}", handler.deleteResourceWidget)
	router.Put("/sites/{siteID}/resources/{resourceID}/widgets/order", handler.reorderResourceWidgets)
	router.Get("/sites/{siteID}/resources/{resourceID}/extensions/{extensionCode}", handler.getResourceExtension)
	router.Patch("/sites/{siteID}/resources/{resourceID}/extensions/{extensionCode}", handler.saveResourceExtension)
	router.Post("/sites/{siteID}/resources/{resourceID}/extensions/{extensionCode}/preview", handler.previewResourceExtension)
	router.Post("/sites/{siteID}/resources/{resourceID}/move", handler.moveResource)
	router.Delete("/sites/{siteID}/resources/{resourceID}", handler.deleteResource)
	router.Post("/sites/{siteID}/resources/{resourceID}/restore", handler.restoreResource)
	router.Delete("/sites/{siteID}/resources/{resourceID}/permanent", handler.deleteResourcePermanent)
	router.Get("/sites/{siteID}/menu", handler.menu)
}

func (h *contentHTTP) listSites(response http.ResponseWriter, request *http.Request) {
	page, perPage, ok := parsePagination(response, request)
	if !ok {
		return
	}
	result, err := h.sites.ListSites(
		request.Context(), actor(request), request.URL.Query().Get("search"), page, perPage,
	)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) menu(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	result, err := h.resources.Menu(request.Context(), actor(request), id)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) listSiteOptions(response http.ResponseWriter, request *http.Request) {
	page, perPage, ok := parsePagination(response, request)
	if !ok {
		return
	}
	result, err := h.sites.ListSiteOptions(
		request.Context(), actor(request), request.URL.Query().Get("search"), page, perPage,
	)
	writeResult(response, http.StatusOK, result, err)
}

type createSiteRequest struct {
	ProfileCode kernel.ProfileCode `json:"profile_code"`
	Domain      string             `json:"domain"`
	Locale      string             `json:"locale"`
	Settings    map[string]any     `json:"settings"`
	IsPublic    bool               `json:"is_public"`
}

func (h *contentHTTP) createSite(response http.ResponseWriter, request *http.Request) {
	var payload createSiteRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.Settings == nil {
		payload.Settings = map[string]any{}
	}
	result, err := h.sites.CreateSite(request.Context(), actor(request), SiteCreateInput{
		ProfileCode: payload.ProfileCode,
		Domain:      payload.Domain,
		Locale:      payload.Locale,
		Settings:    payload.Settings,
		IsPublic:    payload.IsPublic,
	})
	writeResult(response, http.StatusCreated, result, err)
}

func (h *contentHTTP) getSite(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	result, err := h.sites.Site(request.Context(), actor(request), id)
	writeResult(response, http.StatusOK, result, err)
}

type updateSiteRequest struct {
	ProfileCode kernel.ProfileCode `json:"profile_code"`
	Domain      string             `json:"domain"`
	Locale      string             `json:"locale"`
	Settings    map[string]any     `json:"settings"`
	IsPublic    *bool              `json:"is_public"`
}

func (h *contentHTTP) updateSite(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	var payload updateSiteRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.IsPublic == nil {
		writeValidation(response, "is_public is required")
		return
	}
	if payload.Settings == nil {
		writeValidation(response, "settings is required")
		return
	}
	result, err := h.sites.UpdateSite(request.Context(), actor(request), id, SiteUpdateInput{
		ProfileCode: payload.ProfileCode,
		Domain:      payload.Domain,
		Locale:      payload.Locale,
		Settings:    payload.Settings,
		IsPublic:    *payload.IsPublic,
	})
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) deleteSite(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	if err := h.sites.DeleteSite(request.Context(), actor(request), id); err != nil {
		writeManagementError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (h *contentHTTP) listProfiles(response http.ResponseWriter, request *http.Request) {
	result, err := h.sites.Profiles(request.Context(), actor(request))
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) listResourceChildren(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	var parentID *resource.ID
	if raw := request.URL.Query().Get("parent_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			writeBadRequest(response, "parent_id is invalid")
			return
		}
		value := resource.ID(parsed)
		parentID = &value
	}
	result, err := h.resources.ResourceChildren(request.Context(), actor(request), id, parentID)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) resourceMetadata(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	result, err := h.resources.ResourceMetadata(request.Context(), actor(request), id)
	writeResult(response, http.StatusOK, result, err)
}

type createResourceRequest struct {
	ParentID    *resource.ID      `json:"parent_id"`
	Type        resourcetype.Code `json:"type"`
	Template    *template.Code    `json:"template_code"`
	Title       string            `json:"title"`
	MenuTitle   string            `json:"menu_title"`
	Slug        string            `json:"slug"`
	ExternalURL *string           `json:"external_url"`
	Settings    map[string]any    `json:"settings"`
}

func (h *contentHTTP) createResource(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	var payload createResourceRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.Settings == nil {
		writeValidation(response, "settings is required")
		return
	}
	result, err := h.resources.CreateResource(request.Context(), actor(request), id, ResourceCreateInput{
		ParentID:    payload.ParentID,
		Type:        payload.Type,
		Template:    payload.Template,
		Title:       payload.Title,
		MenuTitle:   payload.MenuTitle,
		Slug:        payload.Slug,
		ExternalURL: payload.ExternalURL,
		Settings:    payload.Settings,
	})
	writeResult(response, http.StatusCreated, result, err)
}

func (h *contentHTTP) resourceOptions(response http.ResponseWriter, request *http.Request) {
	siteID, ok := siteID(response, request)
	if !ok {
		return
	}
	result, err := h.resources.ResourceOptions(request.Context(), actor(request), siteID)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) resourceLookup(response http.ResponseWriter, request *http.Request) {
	siteID, ok := siteID(response, request)
	if !ok {
		return
	}
	page, perPage, ok := parsePagination(response, request)
	if !ok {
		return
	}
	result, err := h.resources.ResourceLookup(request.Context(), actor(request), siteID, request.URL.Query().Get("search"), page, perPage)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) getResource(response http.ResponseWriter, request *http.Request) {
	siteID, ok := siteID(response, request)
	if !ok {
		return
	}
	resourceID, ok := resourceID(response, request)
	if !ok {
		return
	}
	result, err := h.resources.Resource(request.Context(), actor(request), siteID, resourceID)
	writeResult(response, http.StatusOK, result, err)
}

type updateResourceRequest struct {
	ParentID      *resource.ID      `json:"parent_id"`
	Type          resourcetype.Code `json:"type"`
	Template      *template.Code    `json:"template_code"`
	Title         string            `json:"title"`
	MenuTitle     string            `json:"menu_title"`
	Slug          string            `json:"slug"`
	Annotation    string            `json:"annotation"`
	Content       string            `json:"content"`
	ContentType   *string           `json:"content_type"`
	ExternalURL   *string           `json:"external_url"`
	IsPublic      *bool             `json:"is_public"`
	IsSearchable  *bool             `json:"is_searchable"`
	InMenu        *bool             `json:"in_menu"`
	InSitemap     *bool             `json:"in_sitemap"`
	Sort          *int              `json:"sort"`
	PublishedAt   *time.Time        `json:"published_at"`
	UnpublishedAt *time.Time        `json:"unpublished_at"`
	Settings      map[string]any    `json:"settings"`
}

func (h *contentHTTP) updateResource(response http.ResponseWriter, request *http.Request) {
	siteID, ok := siteID(response, request)
	if !ok {
		return
	}
	resourceID, ok := resourceID(response, request)
	if !ok {
		return
	}
	var payload updateResourceRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.IsPublic == nil || payload.IsSearchable == nil ||
		payload.InMenu == nil || payload.InSitemap == nil || payload.Sort == nil ||
		payload.Settings == nil {
		writeValidation(response, "resource flags, sort and settings are required")
		return
	}
	result, err := h.resources.UpdateResource(
		request.Context(), actor(request), siteID, resourceID,
		ResourceUpdateInput{
			ParentID:      payload.ParentID,
			Type:          payload.Type,
			Template:      payload.Template,
			Title:         payload.Title,
			MenuTitle:     payload.MenuTitle,
			Slug:          payload.Slug,
			Annotation:    payload.Annotation,
			Content:       payload.Content,
			ContentType:   payload.ContentType,
			ExternalURL:   payload.ExternalURL,
			IsPublic:      *payload.IsPublic,
			IsSearchable:  *payload.IsSearchable,
			InMenu:        *payload.InMenu,
			InSitemap:     *payload.InSitemap,
			Sort:          *payload.Sort,
			PublishedAt:   payload.PublishedAt,
			UnpublishedAt: payload.UnpublishedAt,
			Settings:      payload.Settings,
		},
	)
	writeResult(response, http.StatusOK, result, err)
}

type resourceWidgetPresentationRequest struct {
	View         widget.ViewCode `json:"view"`
	Columns      int             `json:"columns"`
	MarginTop    int             `json:"margin_top"`
	MarginBottom int             `json:"margin_bottom"`
	Enabled      *bool           `json:"enabled"`
	Params       map[string]any  `json:"params"`
}

type createResourceWidgetRequest struct {
	Code widget.Code     `json:"code"`
	Area widget.AreaCode `json:"area"`
	resourceWidgetPresentationRequest
}

func (h *contentHTTP) createResourceWidget(response http.ResponseWriter, request *http.Request) {
	siteID, resourceID, ok := resourceWidgetResourceParams(response, request)
	if !ok {
		return
	}
	var payload createResourceWidgetRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.Params == nil {
		writeValidation(response, "widget params are required")
		return
	}
	result, err := h.resources.CreateResourceWidget(request.Context(), actor(request), siteID, resourceID, resource.CreateWidgetInput{
		Code: payload.Code, Area: payload.Area, View: payload.View, Columns: payload.Columns,
		MarginTop: payload.MarginTop, MarginBottom: payload.MarginBottom,
		Enabled: payload.Enabled, Params: payload.Params,
	})
	writeResult(response, http.StatusCreated, result, err)
}

func (h *contentHTTP) updateResourceWidget(response http.ResponseWriter, request *http.Request) {
	siteID, resourceID, ok := resourceWidgetResourceParams(response, request)
	if !ok {
		return
	}
	bindingID, ok := widgetID(response, request)
	if !ok {
		return
	}
	var payload resourceWidgetPresentationRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.Params == nil || payload.Enabled == nil {
		writeValidation(response, "widget params and enabled are required")
		return
	}
	result, err := h.resources.UpdateResourceWidget(request.Context(), actor(request), siteID, resourceID, bindingID, resource.UpdateWidgetInput{
		View: payload.View, Columns: payload.Columns, MarginTop: payload.MarginTop,
		MarginBottom: payload.MarginBottom, Enabled: payload.Enabled, Params: payload.Params,
	})
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) deleteResourceWidget(response http.ResponseWriter, request *http.Request) {
	siteID, resourceID, ok := resourceWidgetResourceParams(response, request)
	if !ok {
		return
	}
	bindingID, ok := widgetID(response, request)
	if !ok {
		return
	}
	if err := h.resources.DeleteResourceWidget(request.Context(), actor(request), siteID, resourceID, bindingID); err != nil {
		writeManagementError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

type reorderResourceWidgetsRequest struct {
	Items []widget.Order `json:"items"`
}

func (h *contentHTTP) reorderResourceWidgets(response http.ResponseWriter, request *http.Request) {
	siteID, resourceID, ok := resourceWidgetResourceParams(response, request)
	if !ok {
		return
	}
	var payload reorderResourceWidgetsRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.Items == nil {
		writeValidation(response, "widget order items are required")
		return
	}
	result, err := h.resources.ReorderResourceWidgets(request.Context(), actor(request), siteID, resourceID, payload.Items)
	writeResult(response, http.StatusOK, struct {
		Items []ResourceWidget `json:"items"`
	}{Items: result}, err)
}

func resourceWidgetResourceParams(response http.ResponseWriter, request *http.Request) (site.ID, resource.ID, bool) {
	siteID, ok := siteID(response, request)
	if !ok {
		return 0, 0, false
	}
	resourceID, ok := resourceID(response, request)
	if !ok {
		return 0, 0, false
	}
	return siteID, resourceID, true
}

func widgetID(response http.ResponseWriter, request *http.Request) (widget.BindingID, bool) {
	parsed, err := strconv.ParseInt(chi.URLParam(request, "widgetID"), 10, 64)
	if err != nil || parsed <= 0 {
		writeBadRequest(response, "widget_id is invalid")
		return 0, false
	}
	return widget.BindingID(parsed), true
}

type moveResourceRequest struct {
	ParentID *resource.ID `json:"parent_id"`
	Position *int         `json:"position"`
}

func (h *contentHTTP) moveResource(response http.ResponseWriter, request *http.Request) {
	siteID, ok := siteID(response, request)
	if !ok {
		return
	}
	resourceID, ok := resourceID(response, request)
	if !ok {
		return
	}
	var payload moveResourceRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.Position == nil || *payload.Position < 0 {
		writeValidation(response, "position is required")
		return
	}
	result, err := h.resources.MoveResource(request.Context(), actor(request), siteID, resourceID, payload.ParentID, *payload.Position)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) getResourceExtension(
	response http.ResponseWriter,
	request *http.Request,
) {
	siteID, resourceID, code, ok := resourceExtensionParams(response, request)
	if !ok {
		return
	}
	result, err := h.resources.ResourceExtension(
		request.Context(), actor(request), siteID, resourceID, code,
	)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) saveResourceExtension(
	response http.ResponseWriter,
	request *http.Request,
) {
	siteID, resourceID, code, ok := resourceExtensionParams(response, request)
	if !ok {
		return
	}
	var payload json.RawMessage
	if !decodeBody(response, request, &payload) {
		return
	}
	result, err := h.resources.SaveResourceExtension(
		request.Context(), actor(request), siteID, resourceID, code, payload,
	)
	writeResult(response, http.StatusOK, result, err)
}

func (h *contentHTTP) previewResourceExtension(
	response http.ResponseWriter,
	request *http.Request,
) {
	siteID, resourceID, code, ok := resourceExtensionParams(response, request)
	if !ok {
		return
	}
	var payload json.RawMessage
	if !decodeBody(response, request, &payload) {
		return
	}
	result, err := h.resources.PreviewResourceExtension(
		request.Context(), actor(request), siteID, resourceID, code, payload,
	)
	writeResult(response, http.StatusOK, result, err)
}

func resourceExtensionParams(
	response http.ResponseWriter,
	request *http.Request,
) (site.ID, resource.ID, resourceextension.Code, bool) {
	siteID, ok := siteID(response, request)
	if !ok {
		return 0, 0, "", false
	}
	resourceID, ok := resourceID(response, request)
	if !ok {
		return 0, 0, "", false
	}
	code := resourceextension.Code(chi.URLParam(request, "extensionCode"))
	if code == "" {
		writeBadRequest(response, "extension code is invalid")
		return 0, 0, "", false
	}
	return siteID, resourceID, code, true
}

func (h *contentHTTP) deleteResource(response http.ResponseWriter, request *http.Request) {
	h.resourceDelete(response, request, false)
}

func (h *contentHTTP) deleteResourcePermanent(response http.ResponseWriter, request *http.Request) {
	h.resourceDelete(response, request, true)
}

func (h *contentHTTP) resourceDelete(response http.ResponseWriter, request *http.Request, permanent bool) {
	siteID, ok := siteID(response, request)
	if !ok {
		return
	}
	resourceID, ok := resourceID(response, request)
	if !ok {
		return
	}
	err := h.resources.DeleteResource(request.Context(), actor(request), siteID, resourceID, permanent)
	if err != nil {
		writeManagementError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

type restoreResourceRequest struct {
	WithDescendants bool `json:"with_descendants"`
}

func (h *contentHTTP) restoreResource(response http.ResponseWriter, request *http.Request) {
	siteID, ok := siteID(response, request)
	if !ok {
		return
	}
	resourceID, ok := resourceID(response, request)
	if !ok {
		return
	}
	var payload restoreResourceRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	err := h.resources.RestoreResource(request.Context(), actor(request), siteID, resourceID, payload.WithDescendants)
	if err != nil {
		writeManagementError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func actor(request *http.Request) security.Actor {
	current, _ := httptransport.ActorFromContext(request.Context())
	return current
}

func siteID(response http.ResponseWriter, request *http.Request) (site.ID, bool) {
	parsed, err := strconv.ParseInt(chi.URLParam(request, "siteID"), 10, 64)
	if err != nil || parsed <= 0 {
		writeBadRequest(response, "site_id is invalid")
		return 0, false
	}
	return site.ID(parsed), true
}

func resourceID(response http.ResponseWriter, request *http.Request) (resource.ID, bool) {
	parsed, err := strconv.ParseInt(chi.URLParam(request, "resourceID"), 10, 64)
	if err != nil || parsed <= 0 {
		writeBadRequest(response, "resource_id is invalid")
		return 0, false
	}
	return resource.ID(parsed), true
}

func parsePagination(response http.ResponseWriter, request *http.Request) (int, int, bool) {
	page, perPage := 1, 10
	var err error
	if raw := request.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 {
			writeBadRequest(response, "page is invalid")
			return 0, 0, false
		}
	}
	if raw := request.URL.Query().Get("per_page"); raw != "" {
		perPage, err = strconv.Atoi(raw)
		if err != nil || perPage < 1 || perPage > 100 {
			writeBadRequest(response, "per_page is invalid")
			return 0, 0, false
		}
	}
	return page, perPage, true
}

func decodeBody(response http.ResponseWriter, request *http.Request, target any) bool {
	request.Body = http.MaxBytesReader(response, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeBadRequest(response, "request body is invalid")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeBadRequest(response, "request body contains trailing data")
		return false
	}
	return true
}

func writeResult(response http.ResponseWriter, status int, result any, err error) {
	if err != nil {
		writeManagementError(response, err)
		return
	}
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(result)
}

func writeManagementError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, security.ErrUnauthenticated):
		writeUnauthorized(response)
	case errors.Is(err, security.ErrForbidden):
		httptransport.WriteJSONError(response, http.StatusForbidden, "forbidden", "operation is forbidden")
	case errors.Is(err, site.ErrNotFound), errors.Is(err, resource.ErrNotFound), errors.Is(err, file.ErrNotFound), errors.Is(err, file.ErrFolderNotFound), errors.Is(err, file.ErrStorageNotFound):
		httptransport.WriteJSONError(response, http.StatusNotFound, "not_found", "requested object was not found")
	case errors.Is(err, site.ErrConflict), errors.Is(err, resource.ErrConflict), errors.Is(err, file.ErrConflict):
		httptransport.WriteJSONError(response, http.StatusConflict, "conflict", "object conflicts with existing data")
	case errors.Is(err, file.ErrInUse):
		httptransport.WriteJSONError(response, http.StatusConflict, "file_in_use", "file is used by content")
	case errors.Is(err, file.ErrStorageMismatch):
		httptransport.WriteJSONError(response, http.StatusConflict, "storage_mismatch", "items cannot be moved between disks")
	case errors.Is(err, file.ErrInvalidTree):
		httptransport.WriteJSONError(response, http.StatusConflict, "invalid_tree", "folder cannot be moved into itself")
	case errors.Is(err, resource.ErrInvalidTree):
		httptransport.WriteJSONError(response, http.StatusConflict, "invalid_tree", "resource tree operation is invalid")
	case errors.Is(err, resource.ErrReferenced):
		httptransport.WriteJSONError(response, http.StatusConflict, "resource_referenced", "resource is referenced")
	case errors.Is(err, file.ErrInvalidReference):
		writeValidation(response, "file reference is invalid")
	case errors.Is(err, ErrValidation):
		var validation ValidationError
		if errors.As(err, &validation) && len(validation.Fields) > 0 {
			writeFieldValidation(response, validation)
			return
		}
		writeValidation(response, err.Error())
	default:
		httptransport.WriteJSONError(response, http.StatusInternalServerError, "internal_error", "operation failed")
	}
}

func writeBadRequest(response http.ResponseWriter, message string) {
	httptransport.WriteJSONError(response, http.StatusBadRequest, "bad_request", message)
}

func writeUnauthorized(response http.ResponseWriter) {
	response.Header().Set("WWW-Authenticate", "Bearer")
	httptransport.WriteJSONError(
		response,
		http.StatusUnauthorized,
		"unauthenticated",
		"authentication required",
	)
}

func writeValidation(response http.ResponseWriter, message string) {
	httptransport.WriteJSONError(response, http.StatusUnprocessableEntity, "validation_failed", message)
}

func writeFieldValidation(response http.ResponseWriter, validation ValidationError) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(response).Encode(struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details struct {
				Fields []FieldValidationError `json:"fields"`
			} `json:"details"`
		} `json:"error"`
	}{Error: struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details struct {
			Fields []FieldValidationError `json:"fields"`
		} `json:"details"`
	}{
		Code:    "validation_failed",
		Message: validation.Message,
		Details: struct {
			Fields []FieldValidationError `json:"fields"`
		}{Fields: validation.Fields},
	}})
}
