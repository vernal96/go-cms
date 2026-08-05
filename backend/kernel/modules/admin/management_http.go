package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type managementHTTP struct {
	management *Management
}

func registerManagementRoutes(router chi.Router, management *Management) {
	handler := &managementHTTP{management: management}
	router.Get("/sites/options", handler.listSiteOptions)
	router.Get("/sites", handler.listSites)
	router.Post("/sites", handler.createSite)
	router.Get("/site-profiles", handler.listProfiles)
	router.Get("/sites/{siteID}", handler.getSite)
	router.Patch("/sites/{siteID}", handler.updateSite)
	router.Delete("/sites/{siteID}", handler.deleteSite)
	router.Get("/sites/{siteID}/resources", handler.listResourceChildren)
	router.Post("/sites/{siteID}/resources", handler.createResource)
	router.Get("/sites/{siteID}/resource-metadata", handler.resourceMetadata)
}

func (h *managementHTTP) listSites(response http.ResponseWriter, request *http.Request) {
	page, perPage, ok := parsePagination(response, request)
	if !ok {
		return
	}
	result, err := h.management.ListSites(
		request.Context(), actor(request), request.URL.Query().Get("search"), page, perPage,
	)
	writeResult(response, http.StatusOK, result, err)
}

func (h *managementHTTP) listSiteOptions(response http.ResponseWriter, request *http.Request) {
	page, perPage, ok := parsePagination(response, request)
	if !ok {
		return
	}
	result, err := h.management.ListSiteOptions(
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

func (h *managementHTTP) createSite(response http.ResponseWriter, request *http.Request) {
	var payload createSiteRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	if payload.Settings == nil {
		payload.Settings = map[string]any{}
	}
	result, err := h.management.CreateSite(request.Context(), actor(request), SiteCreateInput{
		ProfileCode: payload.ProfileCode,
		Domain:      payload.Domain,
		Locale:      payload.Locale,
		Settings:    payload.Settings,
		IsPublic:    payload.IsPublic,
	})
	writeResult(response, http.StatusCreated, result, err)
}

func (h *managementHTTP) getSite(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	result, err := h.management.Site(request.Context(), actor(request), id)
	writeResult(response, http.StatusOK, result, err)
}

type updateSiteRequest struct {
	ProfileCode kernel.ProfileCode `json:"profile_code"`
	Domain      string             `json:"domain"`
	Locale      string             `json:"locale"`
	IsPublic    *bool              `json:"is_public"`
}

func (h *managementHTTP) updateSite(response http.ResponseWriter, request *http.Request) {
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
	result, err := h.management.UpdateSite(request.Context(), actor(request), id, SiteUpdateInput{
		ProfileCode: payload.ProfileCode,
		Domain:      payload.Domain,
		Locale:      payload.Locale,
		IsPublic:    *payload.IsPublic,
	})
	writeResult(response, http.StatusOK, result, err)
}

func (h *managementHTTP) deleteSite(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	if err := h.management.DeleteSite(request.Context(), actor(request), id); err != nil {
		writeManagementError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (h *managementHTTP) listProfiles(response http.ResponseWriter, request *http.Request) {
	result, err := h.management.Profiles(request.Context(), actor(request))
	writeResult(response, http.StatusOK, result, err)
}

func (h *managementHTTP) listResourceChildren(response http.ResponseWriter, request *http.Request) {
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
	result, err := h.management.ResourceChildren(request.Context(), actor(request), id, parentID)
	writeResult(response, http.StatusOK, result, err)
}

func (h *managementHTTP) resourceMetadata(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	result, err := h.management.ResourceMetadata(request.Context(), actor(request), id)
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
}

func (h *managementHTTP) createResource(response http.ResponseWriter, request *http.Request) {
	id, ok := siteID(response, request)
	if !ok {
		return
	}
	var payload createResourceRequest
	if !decodeBody(response, request, &payload) {
		return
	}
	result, err := h.management.CreateResource(request.Context(), actor(request), id, ResourceCreateInput{
		ParentID:    payload.ParentID,
		Type:        payload.Type,
		Template:    payload.Template,
		Title:       payload.Title,
		MenuTitle:   payload.MenuTitle,
		Slug:        payload.Slug,
		ExternalURL: payload.ExternalURL,
	})
	writeResult(response, http.StatusCreated, result, err)
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
	case errors.Is(err, site.ErrNotFound), errors.Is(err, resource.ErrNotFound):
		httptransport.WriteJSONError(response, http.StatusNotFound, "not_found", "requested object was not found")
	case errors.Is(err, site.ErrConflict), errors.Is(err, resource.ErrConflict):
		httptransport.WriteJSONError(response, http.StatusConflict, "conflict", "object conflicts with existing data")
	case errors.Is(err, ErrValidation):
		writeValidation(response, err.Error())
	default:
		httptransport.WriteJSONError(response, http.StatusInternalServerError, "internal_error", "operation failed")
	}
}

func writeBadRequest(response http.ResponseWriter, message string) {
	httptransport.WriteJSONError(response, http.StatusBadRequest, "bad_request", message)
}

func writeValidation(response http.ResponseWriter, message string) {
	httptransport.WriteJSONError(response, http.StatusUnprocessableEntity, "validation_failed", message)
}
