package forms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type paginationDTO struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type fieldOptionsDTO struct {
	Step        *float64       `json:"step,omitempty"`
	Choices     []field.Choice `json:"choices,omitempty"`
	Multiple    *bool          `json:"multiple,omitempty"`
	Pattern     *string        `json:"pattern,omitempty"`
	MIMETypes   []string       `json:"mime_types,omitempty"`
	MaxFileSize *int64         `json:"max_file_size,omitempty"`
	MaxFiles    *int           `json:"max_files,omitempty"`
	Provider    *string        `json:"provider,omitempty"`
	Text        *string        `json:"text,omitempty"`
	URL         *string        `json:"url,omitempty"`
}

type fieldPayload struct {
	Code           string             `json:"code"`
	Type           field.TypeCode     `json:"type"`
	Label          string             `json:"label"`
	Required       bool               `json:"required"`
	Rules          []string           `json:"rules"`
	Options        json.RawMessage    `json:"options,omitempty"`
	Editor         field.EditorCode   `json:"editor,omitempty"`
	VisibleWhen    *field.VisibleWhen `json:"visible_when,omitempty"`
	ResultLabel    string             `json:"result_label"`
	ShowInResults  bool               `json:"show_in_results"`
	ResultPosition int                `json:"result_position"`
}

type fieldResponse struct {
	ID     FieldID `json:"id"`
	FormID FormID  `json:"form_id"`
	fieldPayload
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type editorResponse struct {
	Form         Form                  `json:"form"`
	Fields       []fieldResponse       `json:"fields"`
	Elements     []Element             `json:"elements"`
	Layout       []LayoutNode          `json:"layout"`
	Statuses     []Status              `json:"statuses"`
	Actions      []Action              `json:"actions"`
	FieldTypes   []field.TypeCode      `json:"available_field_types"`
	ElementTypes []ElementTypeMetadata `json:"available_element_types"`
	ActionTypes  []ActionTypeMetadata  `json:"available_action_types"`
}

func NewManagementHTTPHandler(service *Service) (http.Handler, error) {
	if service == nil {
		return nil, errors.New("Forms service is nil")
	}
	h := &formsHTTP{service: service}
	router := chi.NewRouter()
	router.Get("/forms", h.listForms)
	router.Post("/forms", h.createForm)
	router.Get("/forms/{formID}", h.getForm)
	router.Patch("/forms/{formID}", h.updateForm)
	router.Patch("/forms/{formID}/enabled", h.setFormEnabled)
	router.Delete("/forms/{formID}", h.deleteForm)
	router.Get("/forms/{formID}/editor", h.editor)
	router.Post("/forms/{formID}/fields", h.createField)
	router.Patch("/forms/{formID}/fields/{fieldID}", h.updateField)
	router.Delete("/forms/{formID}/fields/{fieldID}", h.deleteField)
	router.Post("/forms/{formID}/elements", h.createElement)
	router.Patch("/forms/{formID}/elements/{elementID}", h.updateElement)
	router.Delete("/forms/{formID}/elements/{elementID}", h.deleteElement)
	router.Post("/forms/{formID}/containers", h.createContainer)
	router.Put("/forms/{formID}/layout", h.replaceLayout)
	router.Post("/forms/{formID}/statuses", h.createStatus)
	router.Patch("/forms/{formID}/statuses/{statusID}", h.updateStatus)
	router.Delete("/forms/{formID}/statuses/{statusID}", h.deleteStatus)
	router.Post("/forms/{formID}/actions", h.createAction)
	router.Patch("/forms/{formID}/actions/{actionID}", h.updateAction)
	router.Delete("/forms/{formID}/actions/{actionID}", h.deleteAction)
	router.Get("/results", h.listResults)
	router.Get("/results/{resultID}", h.getResult)
	router.Patch("/results/{resultID}/status", h.changeResultStatus)
	router.Delete("/results/{resultID}", h.deleteResult)
	return httptransport.RequireAuthenticated(router), nil
}

type formsHTTP struct{ service *Service }

func (h *formsHTTP) actor(response http.ResponseWriter, request *http.Request) (security.Actor, bool) {
	actor, exists := httptransport.ActorFromContext(request.Context())
	if !exists {
		httptransport.WriteJSONError(response, http.StatusInternalServerError, "internal_error", "request actor is unavailable")
		return security.Actor{}, false
	}
	return actor, true
}

func (h *formsHTTP) listForms(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	query, err := pageQuery(request)
	if err == nil {
		query.Search = request.URL.Query().Get("search")
	}
	var page FormSummaryPage
	if err == nil {
		page, err = h.service.ListForms(request.Context(), actor, query)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, struct {
		Items      []Form        `json:"items"`
		Pagination paginationDTO `json:"pagination"`
	}{page.Items, paginationDTO{query.Page, query.PerPage, page.Total}})
}

type formPayload struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

func (p formPayload) form() Form {
	return Form{Code: p.Code, Name: p.Name, Description: p.Description, Enabled: p.Enabled}
}

func (h *formsHTTP) createForm(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	var payload formPayload
	err := decodeJSONRequest(request, &payload)
	var result FormDetail
	if err == nil {
		result, err = h.service.CreateForm(request.Context(), actor, payload.form())
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, result.Form)
}

func (h *formsHTTP) getForm(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	id, err := pathID[FormID](request, "formID")
	var result FormDetail
	if err == nil {
		result, err = h.service.FormDetail(request.Context(), actor, id)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, result.Form)
}

func (h *formsHTTP) updateForm(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	id, err := pathID[FormID](request, "formID")
	var payload formPayload
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var result Form
	if err == nil {
		item := payload.form()
		item.ID = id
		result, err = h.service.UpdateForm(request.Context(), actor, item)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (h *formsHTTP) setFormEnabled(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	id, err := pathID[FormID](request, "formID")
	var payload struct {
		Enabled bool `json:"enabled"`
	}
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var result Form
	if err == nil {
		result, err = h.service.SetFormEnabled(request.Context(), actor, id, payload.Enabled)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (h *formsHTTP) deleteForm(response http.ResponseWriter, request *http.Request) {
	h.delete(response, request, "formID", func(ctx context.Context, actor security.Actor, id int64) error {
		return h.service.DeleteForm(ctx, actor, FormID(id))
	})
}

func (h *formsHTTP) editor(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	id, err := pathID[FormID](request, "formID")
	var detail FormDetail
	if err == nil {
		detail, err = h.service.FormDetail(request.Context(), actor, id)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	fields := make([]fieldResponse, len(detail.Fields))
	for index, item := range detail.Fields {
		fields[index], err = toFieldResponse(item)
		if err != nil {
			writeManagementError(response, err)
			return
		}
	}
	actionTypes := h.service.AvailableActionTypes()
	availableActions := make(map[string]struct{}, len(actionTypes))
	for _, item := range actionTypes {
		availableActions[item.Code] = struct{}{}
	}
	for _, action := range detail.Actions {
		if _, exists := availableActions[action.ActionType]; exists {
			continue
		}
		actionTypes = append(actionTypes, ActionTypeMetadata{Code: action.ActionType, Label: action.ActionType, Available: false})
		availableActions[action.ActionType] = struct{}{}
	}
	sort.Slice(actionTypes, func(i, j int) bool { return actionTypes[i].Code < actionTypes[j].Code })
	available := make(map[string]struct{}, len(actionTypes))
	for _, item := range actionTypes {
		available[item.Code] = struct{}{}
	}
	for _, action := range detail.Actions {
		if _, exists := available[action.ActionType]; !exists {
			actionTypes = append(actionTypes, ActionTypeMetadata{Code: action.ActionType, Label: action.ActionType, Available: false})
		}
	}
	writeJSON(response, http.StatusOK, editorResponse{detail.Form, fields, detail.Elements, detail.Layout, detail.Statuses, detail.Actions, h.service.AvailableFieldTypes(), h.service.AvailableElementTypes(), actionTypes})
}

func (h *formsHTTP) createField(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var payload fieldPayload
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var item FormField
	if err == nil {
		item, err = payload.field()
	}
	var result FormField
	var node LayoutNode
	if err == nil {
		result, node, err = h.service.CreateField(request.Context(), actor, formID, item)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	converted, err := toFieldResponse(result)
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, struct {
		Field      fieldResponse `json:"field"`
		LayoutNode LayoutNode    `json:"layout_node"`
	}{converted, node})
}

func (h *formsHTTP) updateField(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var fieldID FieldID
	if err == nil {
		fieldID, err = pathID[FieldID](request, "fieldID")
	}
	var payload fieldPayload
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var item FormField
	if err == nil {
		item, err = payload.field()
		item.ID = fieldID
	}
	if err == nil {
		item, err = h.service.UpdateField(request.Context(), actor, formID, item)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	converted, err := toFieldResponse(item)
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, converted)
}

func (h *formsHTTP) deleteField(response http.ResponseWriter, request *http.Request) {
	h.deleteNested(response, request, "fieldID", func(ctx context.Context, actor security.Actor, formID FormID, id int64) error {
		return h.service.DeleteField(ctx, actor, formID, FieldID(id))
	})
}

type elementPayload struct {
	Code   string          `json:"code"`
	Type   ElementTypeCode `json:"type"`
	Config json.RawMessage `json:"config"`
}

func (h *formsHTTP) createElement(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var payload elementPayload
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var item Element
	var node LayoutNode
	if err == nil {
		item, node, err = h.service.CreateElement(request.Context(), actor, formID, Element{Code: payload.Code, Type: payload.Type, Config: payload.Config})
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, struct {
		Element    Element    `json:"element"`
		LayoutNode LayoutNode `json:"layout_node"`
	}{item, node})
}
func (h *formsHTTP) updateElement(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var id ElementID
	if err == nil {
		id, err = pathID[ElementID](request, "elementID")
	}
	var payload elementPayload
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var item Element
	if err == nil {
		item, err = h.service.UpdateElement(request.Context(), actor, formID, Element{ID: id, Code: payload.Code, Type: payload.Type, Config: payload.Config})
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, item)
}
func (h *formsHTTP) deleteElement(response http.ResponseWriter, request *http.Request) {
	h.deleteNested(response, request, "elementID", func(ctx context.Context, actor security.Actor, formID FormID, id int64) error {
		return h.service.DeleteElement(ctx, actor, formID, ElementID(id))
	})
}

func (h *formsHTTP) createContainer(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var payload struct {
		ParentID *LayoutNodeID   `json:"parent_id"`
		Type     ContainerType   `json:"container_type"`
		Position int             `json:"position"`
		Config   json.RawMessage `json:"config"`
	}
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var item LayoutNode
	if err == nil {
		item, err = h.service.CreateContainer(request.Context(), actor, formID, LayoutNode{ParentID: payload.ParentID, ContainerType: payload.Type, Position: payload.Position, Config: payload.Config})
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, item)
}

func (h *formsHTTP) replaceLayout(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var payload struct {
		Nodes []LayoutNode `json:"nodes"`
	}
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var items []LayoutNode
	if err == nil {
		items, err = h.service.ReplaceLayout(request.Context(), actor, formID, payload.Nodes)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, struct {
		Nodes []LayoutNode `json:"nodes"`
	}{items})
}

type statusPayload struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Position  int    `json:"position"`
	IsDefault bool   `json:"is_default"`
}

func (p statusPayload) status() Status {
	return Status{Code: p.Code, Name: p.Name, Color: p.Color, Position: p.Position, IsDefault: p.IsDefault}
}
func (h *formsHTTP) createStatus(response http.ResponseWriter, request *http.Request) {
	h.mutateStatus(response, request, false)
}
func (h *formsHTTP) updateStatus(response http.ResponseWriter, request *http.Request) {
	h.mutateStatus(response, request, true)
}
func (h *formsHTTP) mutateStatus(response http.ResponseWriter, request *http.Request, update bool) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var payload statusPayload
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	item := payload.status()
	if update && err == nil {
		item.ID, err = pathID[StatusID](request, "statusID")
	}
	if err == nil {
		if update {
			item, err = h.service.UpdateStatus(request.Context(), actor, formID, item)
		} else {
			item, err = h.service.CreateStatus(request.Context(), actor, formID, item)
		}
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	status := http.StatusCreated
	if update {
		status = http.StatusOK
	}
	writeJSON(response, status, item)
}
func (h *formsHTTP) deleteStatus(response http.ResponseWriter, request *http.Request) {
	h.deleteNested(response, request, "statusID", func(ctx context.Context, actor security.Actor, formID FormID, id int64) error {
		return h.service.DeleteStatus(ctx, actor, formID, StatusID(id))
	})
}

type actionPayload struct {
	Code       string          `json:"code"`
	Name       string          `json:"name"`
	Enabled    bool            `json:"enabled"`
	Trigger    Trigger         `json:"trigger"`
	ActionType string          `json:"action_type"`
	Config     json.RawMessage `json:"config"`
	Position   int             `json:"position"`
}

func (p actionPayload) action() Action {
	return Action{Code: p.Code, Name: p.Name, Enabled: p.Enabled, Trigger: p.Trigger, ActionType: p.ActionType, Config: p.Config, Position: p.Position}
}
func (h *formsHTTP) createAction(response http.ResponseWriter, request *http.Request) {
	h.mutateAction(response, request, false)
}
func (h *formsHTTP) updateAction(response http.ResponseWriter, request *http.Request) {
	h.mutateAction(response, request, true)
}
func (h *formsHTTP) mutateAction(response http.ResponseWriter, request *http.Request, update bool) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var payload actionPayload
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	item := payload.action()
	if update && err == nil {
		item.ID, err = pathID[ActionID](request, "actionID")
	}
	if err == nil {
		if update {
			item, err = h.service.UpdateAction(request.Context(), actor, formID, item)
		} else {
			item, err = h.service.CreateAction(request.Context(), actor, formID, item)
		}
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	status := http.StatusCreated
	if update {
		status = http.StatusOK
	}
	writeJSON(response, status, item)
}
func (h *formsHTTP) deleteAction(response http.ResponseWriter, request *http.Request) {
	h.deleteNested(response, request, "actionID", func(ctx context.Context, actor security.Actor, formID FormID, id int64) error {
		return h.service.DeleteAction(ctx, actor, formID, ActionID(id))
	})
}

func (h *formsHTTP) listResults(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	page, err := pageQuery(request)
	query := ResultQuery{PageQuery: page}
	if err == nil {
		query.FormID, err = optionalQueryID[FormID](request, "form_id")
	}
	if err == nil {
		query.StatusID, err = optionalQueryID[StatusID](request, "status_id")
	}
	if err == nil {
		query.DateFrom, err = queryTime(request, "date_from")
	}
	if err == nil {
		query.DateTo, err = queryTime(request, "date_to")
	}
	var result ResultSummaryPage
	var columns []FormField
	if err == nil {
		result, columns, err = h.service.ListResults(request.Context(), actor, query)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	columnResponse := make([]fieldResponse, len(columns))
	for index, item := range columns {
		columnResponse[index], err = toFieldResponse(item)
		if err != nil {
			writeManagementError(response, err)
			return
		}
	}
	writeJSON(response, http.StatusOK, struct {
		Items      []ResultSummary `json:"items"`
		Columns    []fieldResponse `json:"columns"`
		Pagination paginationDTO   `json:"pagination"`
	}{result.Items, columnResponse, paginationDTO{query.Page, query.PerPage, result.Total}})
}
func (h *formsHTTP) getResult(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	id, err := pathID[ResultID](request, "resultID")
	var item ResultDetail
	if err == nil {
		item, err = h.service.ResultDetail(request.Context(), actor, id)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, item)
}
func (h *formsHTTP) changeResultStatus(response http.ResponseWriter, request *http.Request) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	id, err := pathID[ResultID](request, "resultID")
	var payload struct {
		StatusID StatusID `json:"status_id"`
	}
	if err == nil {
		err = decodeJSONRequest(request, &payload)
	}
	var item ResultDetail
	if err == nil {
		item, err = h.service.ChangeResultStatus(request.Context(), actor, id, payload.StatusID)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, item)
}
func (h *formsHTTP) deleteResult(response http.ResponseWriter, request *http.Request) {
	h.delete(response, request, "resultID", func(ctx context.Context, actor security.Actor, id int64) error {
		return h.service.DeleteResult(ctx, actor, ResultID(id))
	})
}

func (h *formsHTTP) delete(response http.ResponseWriter, request *http.Request, param string, operation func(context.Context, security.Actor, int64) error) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	id, err := pathID[int64](request, param)
	if err == nil {
		err = operation(request.Context(), actor, id)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}
func (h *formsHTTP) deleteNested(response http.ResponseWriter, request *http.Request, param string, operation func(context.Context, security.Actor, FormID, int64) error) {
	actor, ok := h.actor(response, request)
	if !ok {
		return
	}
	formID, err := pathID[FormID](request, "formID")
	var id int64
	if err == nil {
		id, err = pathID[int64](request, param)
	}
	if err == nil {
		err = operation(request.Context(), actor, formID, id)
	}
	if err != nil {
		writeManagementError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (p fieldPayload) field() (FormField, error) {
	options, err := decodeFieldOptions(p.Type, p.Options)
	if err != nil {
		return FormField{}, err
	}
	return FormField{Code: p.Code, Type: p.Type, Label: p.Label, Required: p.Required, Rules: append([]string(nil), p.Rules...), Options: options, Editor: p.Editor, VisibleWhen: cloneVisibleWhen(p.VisibleWhen), ResultLabel: p.ResultLabel, ShowInResults: p.ShowInResults, ResultPosition: p.ResultPosition}, nil
}

func decodeFieldOptions(code field.TypeCode, raw json.RawMessage) (any, error) {
	if code == field.TypeFile {
		if len(raw) == 0 || string(raw) == "null" {
			return field.FileOptions{}, nil
		}
		var options field.FileOptions
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&options); err != nil {
			return nil, fmt.Errorf("%w: file options are invalid", ErrInvalid)
		}
		return options, nil
	}
	var options *fieldOptionsDTO
	if len(raw) > 0 && string(raw) != "null" {
		options = &fieldOptionsDTO{}
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(options); err != nil {
			if isBuiltInFieldType(code) {
				return nil, fmt.Errorf("%w: field options are invalid", ErrInvalid)
			}
			var custom any
			if json.Unmarshal(raw, &custom) != nil {
				return nil, fmt.Errorf("%w: field options are invalid", ErrInvalid)
			}
			return custom, nil
		}
	}
	switch code {
	case field.TypeString, field.TypeTextarea, field.TypeEmail, field.TypeCheckbox, field.TypeJSON:
		if options != nil {
			return nil, fmt.Errorf("%w: field type %q does not support options", ErrInvalid, code)
		}
		return nil, nil
	case field.TypeInteger:
		result := field.IntegerOptions{}
		if options != nil && options.Step != nil {
			if math.Trunc(*options.Step) != *options.Step {
				return nil, ErrInvalid
			}
			value := int64(*options.Step)
			result.Step = &value
		}
		return result, nil
	case field.TypeFloat:
		result := field.FloatOptions{}
		if options != nil {
			result.Step = options.Step
		}
		return result, nil
	case field.TypeRadio:
		if options == nil {
			return nil, ErrInvalid
		}
		return field.RadioOptions{Choices: append([]field.Choice(nil), options.Choices...)}, nil
	case field.TypeSelect:
		if options == nil {
			return nil, ErrInvalid
		}
		return field.SelectOptions{Choices: append([]field.Choice(nil), options.Choices...), Multiple: options.Multiple != nil && *options.Multiple}, nil
	case field.TypePhone:
		result := field.PhoneOptions{}
		if options != nil && options.Pattern != nil {
			result.Pattern = *options.Pattern
		}
		return result, nil
	case FieldTypeCaptcha:
		result := CaptchaOptions{}
		if options != nil && options.Provider != nil {
			result.Provider = *options.Provider
		}
		return result, nil
	case FieldTypeConsent:
		result := ConsentOptions{}
		if options != nil {
			if options.Text != nil {
				result.Text = *options.Text
			}
			if options.URL != nil {
				result.URL = *options.URL
			}
		}
		return result, nil
	case FieldTypeUpload:
		result := UploadOptions{}
		if options != nil {
			result.MIMETypes = append([]string(nil), options.MIMETypes...)
			if options.MaxFileSize != nil {
				result.MaxFileSize = *options.MaxFileSize
			}
			result.Multiple = options.Multiple != nil && *options.Multiple
			if options.MaxFiles != nil {
				result.MaxFiles = *options.MaxFiles
			}
		}
		return result, nil
	default:
		if len(raw) == 0 || string(raw) == "null" {
			return nil, nil
		}
		var custom any
		if json.Unmarshal(raw, &custom) != nil {
			return nil, fmt.Errorf("%w: field options are invalid", ErrInvalid)
		}
		return custom, nil
	}
}

func isBuiltInFieldType(code field.TypeCode) bool {
	switch code {
	case field.TypeString, field.TypeInteger, field.TypeFloat, field.TypeCheckbox, field.TypeRadio, field.TypeSelect,
		field.TypeTextarea, field.TypeEmail, field.TypePhone, field.TypeFile, field.TypeJSON,
		FieldTypeCaptcha, FieldTypeConsent, FieldTypeUpload:
		return true
	default:
		return false
	}
}

func toFieldResponse(item FormField) (fieldResponse, error) {
	options, err := encodeFieldOptions(item.Type, item.Options)
	if err != nil {
		return fieldResponse{}, err
	}
	return fieldResponse{ID: item.ID, FormID: item.FormID, fieldPayload: fieldPayload{Code: item.Code, Type: item.Type, Label: item.Label, Required: item.Required, Rules: append([]string(nil), item.Rules...), Options: options, Editor: item.Editor, VisibleWhen: cloneVisibleWhen(item.VisibleWhen), ResultLabel: item.ResultLabel, ShowInResults: item.ShowInResults, ResultPosition: item.ResultPosition}, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

func encodeFieldOptions(code field.TypeCode, value any) (json.RawMessage, error) {
	marshal := func(options *fieldOptionsDTO) (json.RawMessage, error) {
		if options == nil {
			return nil, nil
		}
		raw, err := json.Marshal(options)
		return json.RawMessage(raw), err
	}
	switch code {
	case field.TypeString, field.TypeTextarea, field.TypeEmail, field.TypeCheckbox, field.TypeJSON:
		if value != nil {
			return nil, ErrInvalid
		}
		return nil, nil
	case field.TypeInteger:
		options, ok := value.(field.IntegerOptions)
		if value == nil {
			ok = true
		}
		if !ok {
			return nil, ErrInvalid
		}
		result := &fieldOptionsDTO{}
		if options.Step != nil {
			step := float64(*options.Step)
			result.Step = &step
		}
		return marshal(result)
	case field.TypeFloat:
		options, ok := value.(field.FloatOptions)
		if value == nil {
			ok = true
		}
		if !ok {
			return nil, ErrInvalid
		}
		return marshal(&fieldOptionsDTO{Step: options.Step})
	case field.TypeRadio:
		options, ok := value.(field.RadioOptions)
		if !ok {
			return nil, ErrInvalid
		}
		return marshal(&fieldOptionsDTO{Choices: append([]field.Choice(nil), options.Choices...)})
	case field.TypeSelect:
		options, ok := value.(field.SelectOptions)
		if !ok {
			return nil, ErrInvalid
		}
		multiple := options.Multiple
		return marshal(&fieldOptionsDTO{Choices: append([]field.Choice(nil), options.Choices...), Multiple: &multiple})
	case field.TypePhone:
		options, ok := value.(field.PhoneOptions)
		if value == nil {
			ok = true
		}
		if !ok {
			return nil, ErrInvalid
		}
		pattern := options.Pattern
		return marshal(&fieldOptionsDTO{Pattern: &pattern})
	case field.TypeFile:
		options, ok := value.(field.FileOptions)
		if value == nil {
			ok = true
		}
		if !ok {
			return nil, ErrInvalid
		}
		return json.Marshal(options)
	case FieldTypeCaptcha:
		options, err := captchaOptions(value)
		if err != nil {
			return nil, err
		}
		provider := options.Provider
		return marshal(&fieldOptionsDTO{Provider: &provider})
	case FieldTypeConsent:
		options, err := consentOptions(value)
		if err != nil {
			return nil, err
		}
		text, url := options.Text, options.URL
		return marshal(&fieldOptionsDTO{Text: &text, URL: &url})
	case FieldTypeUpload:
		options, err := uploadOptions(value)
		if err != nil {
			return nil, err
		}
		multiple, maxSize, maxFiles := options.Multiple, options.MaxFileSize, options.MaxFiles
		return marshal(&fieldOptionsDTO{MIMETypes: append([]string(nil), options.MIMETypes...), Multiple: &multiple, MaxFileSize: &maxSize, MaxFiles: &maxFiles})
	default:
		if value == nil {
			return nil, nil
		}
		raw, err := json.Marshal(value)
		return json.RawMessage(raw), err
	}
}

func pageQuery(request *http.Request) (PageQuery, error) {
	query := PageQuery{}
	for key, target := range map[string]*int{"page": &query.Page, "per_page": &query.PerPage} {
		if raw := request.URL.Query().Get(key); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil {
				return PageQuery{}, ErrInvalid
			}
			*target = value
		}
	}
	return normalizePage(query)
}
func pathID[T ~int64](request *http.Request, key string) (T, error) {
	value, err := strconv.ParseInt(chi.URLParam(request, key), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%w: ID is invalid", ErrInvalid)
	}
	return T(value), nil
}
func optionalQueryID[T ~int64](request *http.Request, key string) (T, error) {
	raw := request.URL.Query().Get(key)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, ErrInvalid
	}
	return T(value), nil
}
func queryTime(request *http.Request, key string) (*time.Time, error) {
	raw := strings.TrimSpace(request.URL.Query().Get(key))
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, ErrInvalid
	}
	value = value.UTC()
	return &value, nil
}
func decodeJSONRequest(request *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 8<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: JSON payload is invalid", ErrInvalid)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing JSON data", ErrInvalid)
	}
	return nil
}
func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeManagementError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, security.ErrUnauthenticated):
		response.Header().Set("WWW-Authenticate", "Bearer")
		httptransport.WriteJSONError(response, http.StatusUnauthorized, "unauthenticated", "authentication required")
	case errors.Is(err, security.ErrForbidden):
		httptransport.WriteJSONError(response, http.StatusForbidden, "forbidden", "Forms access is forbidden")
	case errors.Is(err, ErrNotFound):
		httptransport.WriteJSONError(response, http.StatusNotFound, "not_found", "Forms item not found")
	case errors.Is(err, ErrConflict), errors.Is(err, ErrRuntimeDraining), errors.Is(err, ErrActiveExecutions):
		httptransport.WriteJSONError(response, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrValidation):
		writeValidationEnvelope(response, err)
	default:
		httptransport.WriteJSONError(response, http.StatusInternalServerError, "internal_error", "Forms operation failed")
	}
}

func writeValidationEnvelope(response http.ResponseWriter, err error) {
	details := []map[string]string{}
	var fields FieldValidationErrors
	if errors.As(err, &fields) {
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			for _, rule := range fields[key] {
				details = append(details, map[string]string{"key": key, "rule": rule, "param": ""})
			}
		}
	}
	writeJSON(response, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{"code": "validation_failed", "message": "Forms validation failed", "details": map[string]any{"fields": details}}})
}

// Public profile HTTP.
func newPublicHTTPBuilder(service *Service) httptransport.Builder {
	return httptransport.BuilderFunc(func(context.Context) (httptransport.Contribution, error) {
		if service == nil {
			return httptransport.Contribution{}, errors.New("Forms public service is nil")
		}
		h := &publicFormsHTTP{service: service}
		return httptransport.Contribution{Routes: func(registrar httptransport.Registrar) error {
			if err := registrar.Route(httptransport.Route{Name: "forms.schema", Method: http.MethodGet, Pattern: "/forms/{code}", Handler: http.HandlerFunc(h.schema)}); err != nil {
				return err
			}
			return registrar.Route(httptransport.Route{Name: "forms.submit", Method: http.MethodPost, Pattern: "/forms/{code}/submit", Handler: http.HandlerFunc(h.submit)})
		}}, nil
	})
}

type publicFormsHTTP struct{ service *Service }
type publicField struct {
	Code        string             `json:"code"`
	Type        field.TypeCode     `json:"type"`
	Label       string             `json:"label"`
	Required    bool               `json:"required"`
	Rules       []string           `json:"rules"`
	Options     json.RawMessage    `json:"options,omitempty"`
	Editor      field.EditorCode   `json:"editor,omitempty"`
	VisibleWhen *field.VisibleWhen `json:"visible_when,omitempty"`
	Captcha     map[string]any     `json:"captcha,omitempty"`
}
type publicElement struct {
	Code   string          `json:"code"`
	Type   ElementTypeCode `json:"type"`
	Config any             `json:"config"`
}
type publicLayoutNode struct {
	Key           string          `json:"key"`
	Parent        string          `json:"parent,omitempty"`
	Kind          LayoutKind      `json:"kind"`
	FieldCode     string          `json:"field_code,omitempty"`
	ElementCode   string          `json:"element_code,omitempty"`
	ContainerType ContainerType   `json:"container_type,omitempty"`
	Position      int             `json:"position"`
	Config        json.RawMessage `json:"config,omitempty"`
}

func (h *publicFormsHTTP) schema(response http.ResponseWriter, request *http.Request) {
	detail, err := h.service.PublicForm(request.Context(), chi.URLParam(request, "code"))
	if err != nil {
		writePublicError(response, err)
		return
	}
	fields := make([]publicField, len(detail.Fields))
	for index, item := range detail.Fields {
		options, optionErr := encodeFieldOptions(item.Type, item.Options)
		if optionErr != nil {
			writePublicError(response, optionErr)
			return
		}
		fields[index] = publicField{Code: item.Code, Type: item.Type, Label: item.Label, Required: item.Required, Rules: append([]string(nil), item.Rules...), Options: options, Editor: item.Editor, VisibleWhen: cloneVisibleWhen(item.VisibleWhen)}
		if item.Type == FieldTypeCaptcha {
			fields[index].Captcha, err = h.service.CaptchaPublicConfig(request.Context(), item)
			if err != nil {
				writePublicError(response, err)
				return
			}
		}
	}
	elements := make([]publicElement, len(detail.Elements))
	elementCodes := make(map[ElementID]string, len(detail.Elements))
	for index, item := range detail.Elements {
		elementCodes[item.ID] = item.Code
		var config any
		if item.Type == ElementImage {
			var raw map[string]any
			if json.Unmarshal(item.Config, &raw) != nil {
				writePublicError(response, ErrInvalid)
				return
			}
			url, urlErr := h.service.PublicImageURL(request.Context(), item.Config)
			if urlErr != nil {
				writePublicError(response, ErrNotFound)
				return
			}
			delete(raw, "file_id")
			raw["url"] = url
			config = raw
		} else if json.Unmarshal(item.Config, &config) != nil {
			writePublicError(response, ErrInvalid)
			return
		}
		elements[index] = publicElement{Code: item.Code, Type: item.Type, Config: config}
	}
	fieldCodes := make(map[FieldID]string, len(detail.Fields))
	for _, item := range detail.Fields {
		fieldCodes[item.ID] = item.Code
	}
	keys := make(map[LayoutNodeID]string, len(detail.Layout))
	for index, item := range detail.Layout {
		keys[item.ID] = fmt.Sprintf("n%d", index+1)
	}
	layout := make([]publicLayoutNode, len(detail.Layout))
	for index, item := range detail.Layout {
		node := publicLayoutNode{Key: keys[item.ID], Kind: item.Kind, ContainerType: item.ContainerType, Position: item.Position, Config: item.Config}
		if item.ParentID != nil {
			node.Parent = keys[*item.ParentID]
		}
		if item.FieldID != nil {
			node.FieldCode = fieldCodes[*item.FieldID]
		}
		if item.ElementID != nil {
			node.ElementCode = elementCodes[*item.ElementID]
		}
		layout[index] = node
	}
	writeJSON(response, http.StatusOK, map[string]any{"code": detail.Form.Code, "name": detail.Form.Name, "description": detail.Form.Description, "fields": fields, "elements": elements, "layout": layout})
}

func (h *publicFormsHTTP) submit(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, h.service.limits.MaxRequestSize)
	ctx := request.Context()
	if h.service.limits.SubmissionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.service.limits.SubmissionTimeout)
		defer cancel()
	}
	values := map[string]any{}
	uploads := []UploadInput{}
	closers := []io.Closer{}
	defer func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}()
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(request.Header.Get("Content-Type"), ";")[0]))
	var err error
	if mediaType == "multipart/form-data" {
		if err = request.ParseMultipartForm(64 << 10); err == nil {
			defer request.MultipartForm.RemoveAll()
			values = multipartValues(request.MultipartForm.Value)
			uploads, closers, err = multipartUploads(request.MultipartForm.File)
		}
	} else {
		var payload struct {
			Values map[string]any `json:"values"`
		}
		err = decodeJSONRequest(request, &payload)
		values = payload.Values
		if values == nil {
			values = map[string]any{}
		}
	}
	if err != nil {
		if errors.As(err, new(*http.MaxBytesError)) {
			writePublicError(response, ErrRequestTooLarge)
		} else {
			writePublicError(response, fmt.Errorf("%w: request payload is invalid", ErrInvalid))
		}
		return
	}
	actor, exists := httptransport.ActorFromContext(request.Context())
	if !exists {
		writePublicError(response, errors.New("request actor is unavailable"))
		return
	}
	_, err = h.service.Submit(ctx, actor, SubmitInput{FormCode: chi.URLParam(request, "code"), Values: values, Uploads: uploads, UserAgent: request.UserAgent(), ClientAddress: clientAddress(request)})
	if err != nil {
		writePublicError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]bool{"success": true})
}

func multipartValues(source map[string][]string) map[string]any {
	result := make(map[string]any, len(source))
	for key, items := range source {
		values := make([]any, len(items))
		for index, item := range items {
			var decoded any
			if json.Unmarshal([]byte(item), &decoded) != nil {
				decoded = item
			}
			values[index] = decoded
		}
		if len(values) == 1 {
			result[key] = values[0]
		} else {
			result[key] = values
		}
	}
	return result
}
func multipartUploads(source map[string][]*multipart.FileHeader) ([]UploadInput, []io.Closer, error) {
	result := []UploadInput{}
	closers := []io.Closer{}
	for fieldCode, items := range source {
		for position, item := range items {
			body, err := item.Open()
			if err != nil {
				return nil, closers, err
			}
			closers = append(closers, body)
			result = append(result, UploadInput{FieldCode: fieldCode, Position: position, Filename: item.Filename, MIMEType: item.Header.Get("Content-Type"), Size: item.Size, Body: body})
		}
	}
	return result, closers, nil
}
func clientAddress(request *http.Request) string {
	value := strings.TrimSpace(request.RemoteAddr)
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return value
}

func writePublicError(response http.ResponseWriter, err error) {
	var fields FieldValidationErrors
	switch {
	case errors.As(err, &fields):
		writeJSON(response, http.StatusUnprocessableEntity, map[string]any{"error": "validation_failed", "fields": fields})
	case errors.Is(err, ErrNotFound):
		httptransport.WriteJSONError(response, http.StatusNotFound, "not_found", "form not found")
	case errors.Is(err, ErrRequestTooLarge):
		httptransport.WriteJSONError(response, http.StatusRequestEntityTooLarge, "request_too_large", "form request is too large")
	case errors.Is(err, ErrRateLimited):
		response.Header().Set("Retry-After", "60")
		httptransport.WriteJSONError(response, http.StatusTooManyRequests, "rate_limited", "too many submissions")
	case errors.Is(err, ErrRuntimeDraining):
		httptransport.WriteJSONError(response, http.StatusServiceUnavailable, "unavailable", "form submission is temporarily unavailable")
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrValidation):
		httptransport.WriteJSONError(response, http.StatusUnprocessableEntity, "validation_failed", "form submission is invalid")
	default:
		httptransport.WriteJSONError(response, http.StatusInternalServerError, "internal_error", "form submission failed")
	}
}

var _ = filesystem.VisibilityPublic
