package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/job"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type testFields map[field.TypeCode]field.Type

func (r testFields) FieldType(code field.TypeCode) (field.Type, bool) {
	value, ok := r[code]
	return value, ok
}

func standardFields() testFields {
	result := testFields{}
	for _, item := range field.StandardTypes() {
		result[item.Code()] = item
	}
	return result
}

type testFiles struct {
	items map[file.ID]file.File
	body  map[file.ID]string
	urls  map[file.ID]string
}

func (f *testFiles) GetFile(_ context.Context, _ security.Actor, id file.ID) (file.File, error) {
	item, ok := f.items[id]
	if !ok {
		return file.File{}, file.ErrNotFound
	}
	return item, nil
}
func (f *testFiles) URL(_ context.Context, _ security.Actor, id file.ID) (string, error) {
	value, ok := f.urls[id]
	if !ok {
		return "", errors.New("not public")
	}
	return value, nil
}
func (f *testFiles) Open(_ context.Context, _ security.Actor, id file.ID) (file.OpenedFile, error) {
	item, err := f.GetFile(context.Background(), security.System(), id)
	if err != nil {
		return file.OpenedFile{}, err
	}
	return file.OpenedFile{File: item, Body: io.NopCloser(strings.NewReader(f.body[id]))}, nil
}

func testRenderer(t *testing.T, policy SenderPolicy) (*Renderer, *testFiles) {
	t.Helper()
	files := &testFiles{items: map[file.ID]file.File{7: {ID: 7, Name: "invoice.pdf", MIMEType: "application/pdf", Size: 3, ChecksumSHA256: "abc"}}, body: map[file.ID]string{7: "pdf"}, urls: map[file.ID]string{7: "https://example.com/files/7"}}
	renderer, err := NewRenderer(standardFields(), files, kernel.NewRuntimeScope("5", "example.com", "ru-RU", nil), RendererConfig{SenderPolicy: policy})
	if err != nil {
		t.Fatal(err)
	}
	return renderer, files
}

func mailTemplate() Template {
	return Template{
		SiteID: 5, Code: "welcome", Name: "Welcome", Enabled: true, Transport: "default",
		From:    AddressTemplate{Name: "Site", Email: "noreply@example.com"},
		To:      []AddressTemplate{{Name: "{{data.name}}", Email: "{{data.email}}"}},
		Subject: "Hello {{data.name}}", ContentType: ContentHTML,
		HTMLBody: "<p>{{data.name}} {{data.count}}</p>",
		Variables: []field.Definition{
			{Key: "name", Type: field.TypeString, Label: "Name"},
			{Key: "email", Type: field.TypeEmail, Label: "Email"},
			{Key: "count", Type: field.TypeInteger, Label: "Count"},
			{Key: "invoice", Type: field.TypeFile, Label: "Invoice", Options: field.FileOptions{}},
		},
	}
}

func TestRendererUsesTypedOptionalValuesAndEscapesHTML(t *testing.T) {
	t.Parallel()
	renderer, _ := testRenderer(t, SenderPolicy{AllowedDomains: []string{"example.com"}})
	result, err := renderer.Render(context.Background(), mailTemplate(), map[string]any{"name": `<script>alert(1)</script>`, "email": "person@example.net", "count": float64(2)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Subject != "Hello <script>alert(1)</script>" || !strings.Contains(result.HTMLBody, "&lt;script&gt;") || !strings.Contains(result.HTMLBody, "2") {
		t.Fatalf("rendered = %#v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v", result.Warnings)
	}

	missing, err := renderer.Render(context.Background(), mailTemplate(), map[string]any{"email": "person@example.net"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing.Warnings) != 4 {
		t.Fatalf("missing warnings = %#v", missing.Warnings)
	}
}

func TestRendererRejectsHeadersRecipientsAndSenderSpoofing(t *testing.T) {
	t.Parallel()
	renderer, _ := testRenderer(t, SenderPolicy{AllowedAddresses: []string{"noreply@example.com"}})
	template := mailTemplate()
	for _, test := range []struct {
		name   string
		mutate func(*Template)
		values map[string]any
		target error
	}{
		{"header injection", func(item *Template) { item.Subject = "ok\r\nBcc: bad@example.com" }, map[string]any{"email": "person@example.net"}, ErrInvalid},
		{"invalid recipient", func(*Template) {}, map[string]any{"email": "not-an-email"}, ErrInvalid},
		{"no recipient", func(item *Template) { item.To = []AddressTemplate{{Email: "{{data.name}}"}} }, nil, ErrNoRecipients},
		{"sender policy", func(item *Template) { item.From.Email = "spoof@evil.test" }, map[string]any{"email": "person@example.net"}, ErrSenderNotAllowed},
	} {
		item := template
		test.mutate(&item)
		_, err := renderer.Render(context.Background(), item, test.values)
		if !errors.Is(err, test.target) {
			t.Fatalf("%s error = %v", test.name, err)
		}
	}
}

func TestRendererDefaultsSenderPolicyToCurrentSiteDomain(t *testing.T) {
	t.Parallel()
	renderer, _ := testRenderer(t, SenderPolicy{})
	template := mailTemplate()
	template.From.Email = "sender@other.test"
	_, err := renderer.Render(context.Background(), template, map[string]any{"email": "person@example.net"})
	if !errors.Is(err, ErrSenderNotAllowed) {
		t.Fatalf("sender outside current site domain error = %v", err)
	}
}

func TestRendererResolvesStaticAndVariableAttachments(t *testing.T) {
	t.Parallel()
	renderer, _ := testRenderer(t, SenderPolicy{})
	template := mailTemplate()
	staticID := file.ID(7)
	template.ContentType, template.TextBody, template.HTMLBody = ContentText, "Invoice {{data.invoice}}", ""
	template.Attachments = []AttachmentTemplate{
		{Source: AttachmentStatic, FileID: &staticID, FilenameTemplate: "static-{{data.name}}.pdf"},
		{Source: AttachmentVariable, Variable: "data.invoice"},
	}
	result, err := renderer.Render(context.Background(), template, map[string]any{"name": "Alice", "email": "a@example.net", "invoice": float64(7)})
	if err != nil {
		t.Fatal(err)
	}
	if result.TextBody != "Invoice https://example.com/files/7" || len(result.Attachments) != 2 || result.Attachments[0].Filename != "static-Alice.pdf" || result.Attachments[1].FileID != 7 {
		t.Fatalf("rendered attachments = %#v", result)
	}
	missing, err := renderer.Render(context.Background(), template, map[string]any{"email": "a@example.net"})
	if err != nil {
		t.Fatal(err)
	}
	if len(missing.Attachments) != 1 {
		t.Fatalf("missing variable attachment = %#v", missing)
	}
}

type allowAuthorizer struct{}

func (allowAuthorizer) Check(context.Context, security.Actor, permission.Code) error { return nil }

type memoryRepository struct {
	template  Template
	message   Message
	queued    eventbus.Message
	attempts  []DeliveryAttempt
	claimable bool
	finishErr error
}

func (r *memoryRepository) ListTemplates(context.Context, site.ID, PageQuery) (TemplatePage, error) {
	return TemplatePage{Items: []Template{r.template}, Total: 1}, nil
}
func (r *memoryRepository) ListEnabledTemplates(context.Context, site.ID, PageQuery) (TemplatePage, error) {
	if !r.template.Enabled {
		return TemplatePage{Items: []Template{}}, nil
	}
	return TemplatePage{Items: []Template{r.template}, Total: 1}, nil
}
func (r *memoryRepository) TemplateByID(_ context.Context, siteID site.ID, id TemplateID) (Template, error) {
	if r.template.SiteID != siteID || r.template.ID != id {
		return Template{}, ErrNotFound
	}
	return r.template, nil
}
func (r *memoryRepository) TemplateByCode(_ context.Context, siteID site.ID, code string) (Template, error) {
	if r.template.SiteID != siteID || r.template.Code != code {
		return Template{}, ErrNotFound
	}
	return r.template, nil
}
func (r *memoryRepository) CreateTemplate(_ context.Context, item Template) (Template, error) {
	item.ID = 1
	r.template = item
	return item, nil
}
func (r *memoryRepository) UpdateTemplate(_ context.Context, item Template) (Template, error) {
	r.template = item
	return item, nil
}
func (r *memoryRepository) DeleteTemplate(context.Context, site.ID, TemplateID) error { return nil }
func (r *memoryRepository) CreateMessageAndJob(_ context.Context, item Message, queued eventbus.Message) (Message, error) {
	item.ID = 11
	var envelope job.Envelope
	if err := json.Unmarshal(queued.Body, &envelope); err != nil {
		return Message{}, err
	}
	envelope.Payload, _ = json.Marshal(struct {
		MessageID MessageID `json:"message_id"`
	}{item.ID})
	queued.Body, _ = json.Marshal(envelope)
	r.message, r.queued, r.claimable = item, queued, true
	return item, nil
}
func (r *memoryRepository) ListMessages(context.Context, site.ID, PageQuery) (MessagePage, error) {
	return MessagePage{Items: []Message{r.message}, Total: 1}, nil
}
func (r *memoryRepository) MessageDetail(context.Context, site.ID, MessageID) (MessageDetail, error) {
	return MessageDetail{Message: r.message, Attempts: r.attempts}, nil
}
func (r *memoryRepository) DeleteMessage(context.Context, site.ID, MessageID) error { return nil }
func (r *memoryRepository) ClaimMessage(_ context.Context, siteID site.ID, id MessageID) (Message, DeliveryAttempt, bool, error) {
	if !r.claimable || r.message.ID != id || r.message.SiteID != siteID || r.message.Status == StatusAccepted {
		return Message{}, DeliveryAttempt{}, false, nil
	}
	r.claimable = false
	r.message.Status = StatusSending
	attempt := DeliveryAttempt{MessageID: id, AttemptNumber: len(r.attempts) + 1, Status: AttemptSending, Transport: r.message.Transport}
	r.attempts = append(r.attempts, attempt)
	return r.message, attempt, true, nil
}
func (r *memoryRepository) FinishAttempt(_ context.Context, _ MessageID, number int, result DeliveryResult, sendErr error) error {
	if r.finishErr != nil {
		return r.finishErr
	}
	item := &r.attempts[number-1]
	item.Driver, item.ResponseCode = result.Driver, result.ResponseCode
	if sendErr != nil {
		item.Status, item.SafeError, r.message.Status = AttemptFailed, sendErr.Error(), StatusFailed
		r.claimable = true
	} else {
		item.Status, r.message.Status = AttemptAccepted, StatusAccepted
	}
	return nil
}
func (r *memoryRepository) Cleanup(context.Context, time.Duration, int) (int64, error) { return 0, nil }

func TestPreviewAndQueueShareRendererAndPersistImmutableSnapshot(t *testing.T) {
	t.Parallel()
	renderer, _ := testRenderer(t, SenderPolicy{})
	repository := &memoryRepository{template: mailTemplate()}
	repository.template.ID = 3
	transports, _ := NewTransportRegistry(map[TransportAlias]Transport{"default": NullTransport{}})
	service, err := NewService(5, repository, renderer, allowAuthorizer{}, transports, "default", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]any{"name": "Alice", "email": "a@example.net", "count": float64(2)}
	preview, err := service.Preview(context.Background(), security.User(9), 3, values)
	if err != nil {
		t.Fatal(err)
	}
	message, err := service.QueueManual(context.Background(), security.User(9), ManualSendInput{TemplateID: 3, Values: values, ActorName: "Editor"})
	if err != nil {
		t.Fatal(err)
	}
	if message.Subject != preview.Subject || message.HTMLBody != preview.HTMLBody || message.Status != StatusQueued || message.RequestedBy == nil || *message.RequestedBy != 9 || !strings.HasPrefix(message.RFCMessageID, "<") {
		t.Fatalf("queued = %#v preview=%#v", message, preview)
	}
	repository.template.Subject = "Changed"
	if repository.message.Subject != preview.Subject {
		t.Fatal("queued snapshot changed with template")
	}
	var envelope job.Envelope
	if err := json.Unmarshal(repository.queued.Body, &envelope); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		MessageID MessageID `json:"message_id"`
	}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil || payload.MessageID != message.ID || envelope.ScopeID != "5" {
		t.Fatalf("job payload = %#v, %v", payload, err)
	}
	repository.template.Enabled = false
	if _, err := service.Preview(context.Background(), security.User(9), 3, values); !errors.Is(err, ErrTemplateDisabled) {
		t.Fatalf("disabled preview error = %v", err)
	}
	if _, err := service.QueueByCode(context.Background(), QueueInput{TemplateCode: "welcome", Values: values}); !errors.Is(err, ErrTemplateDisabled) {
		t.Fatalf("disabled automatic queue error = %v", err)
	}
}

type emptySiteCatalog struct{}

func (emptySiteCatalog) RuntimeByID(site.ID) (*site.Runtime, bool) { return nil, false }

type sitePolicy struct {
	allowed site.ID
	seen    site.ID
}

func (p *sitePolicy) Check(_ context.Context, _ security.Actor, siteID site.ID, _ group.SiteAccessAction) error {
	p.seen = siteID
	if siteID != p.allowed {
		return security.ErrForbidden
	}
	return nil
}

func TestMailHTTPRequiresAuthenticationAndChecksSiteAccessBeforeRuntimeResolution(t *testing.T) {
	t.Parallel()
	policy := &sitePolicy{allowed: 5}
	management, err := NewManagement(emptySiteCatalog{}, policy)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHTTPHandler(management)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Mount("/api/sites/{siteID}/mail", handler)

	unauthenticated := httptest.NewRecorder()
	guestRequest := httptest.NewRequest(http.MethodGet, "/api/sites/8/mail/messages", nil)
	guestRequest = guestRequest.WithContext(httptransport.WithActor(guestRequest.Context(), security.Guest()))
	router.ServeHTTP(unauthenticated, guestRequest)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", unauthenticated.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/sites/8/mail/messages", nil)
	request = request.WithContext(httptransport.WithActor(request.Context(), security.User(9)))
	forbidden := httptest.NewRecorder()
	router.ServeHTTP(forbidden, request)
	if forbidden.Code != http.StatusForbidden || policy.seen != 8 || !strings.Contains(forbidden.Body.String(), `"code":"forbidden"`) {
		t.Fatalf("forbidden response = %d %s, site=%d", forbidden.Code, forbidden.Body.String(), policy.seen)
	}
}

func TestTemplateResponseUsesStableLowercaseChoiceDTO(t *testing.T) {
	t.Parallel()
	template := mailTemplate()
	template.Variables = []field.Definition{{Key: "kind", Type: field.TypeSelect, Label: "Kind", Options: field.SelectOptions{Choices: []field.Choice{{Value: "a", Label: "A"}}}}}
	response, err := toTemplateResponse(template)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"choices":[{"value":"a","label":"A"}]`) || strings.Contains(string(raw), `"Value"`) {
		t.Fatalf("response JSON = %s", raw)
	}
}

type countingTransport struct {
	count   int
	failure error
}

func (*countingTransport) Driver() string { return "test" }
func (t *countingTransport) Send(_ context.Context, delivery Delivery) (DeliveryResult, error) {
	t.count++
	if len(delivery.Attachments) > 0 {
		data, _ := io.ReadAll(delivery.Attachments[0].Body)
		if string(data) != "pdf" {
			return DeliveryResult{}, errors.New("wrong attachment")
		}
	}
	return DeliveryResult{Driver: "test", ResponseCode: "250"}, t.failure
}

func TestWorkerRecordsAttemptAndSkipsDuplicateTerminalDelivery(t *testing.T) {
	t.Parallel()
	_, files := testRenderer(t, SenderPolicy{})
	repository := &memoryRepository{message: Message{ID: 11, SiteID: 5, Transport: "default", Status: StatusQueued, Attachments: []Attachment{{FileID: 7}}}, claimable: true}
	transport := &countingTransport{}
	registry, _ := NewTransportRegistry(map[TransportAlias]Transport{"default": transport})
	worker, _ := NewWorker(5, repository, files, registry)
	payload, _ := json.Marshal(struct {
		MessageID MessageID `json:"message_id"`
	}{11})
	item := job.Envelope{ID: "01H00000000000000000000000", Name: SendJobName, ScopeID: "5", SchemaVersion: 1, Payload: payload}
	if err := worker.Handle(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	if err := worker.Handle(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	if transport.count != 1 || repository.message.Status != StatusAccepted || len(repository.attempts) != 1 || repository.attempts[0].ResponseCode != "250" {
		t.Fatalf("worker state = count %d message %#v attempts %#v", transport.count, repository.message, repository.attempts)
	}
}

func TestWorkerRecordsFailureThenRetryAndRejectsWrongSiteScope(t *testing.T) {
	t.Parallel()
	_, files := testRenderer(t, SenderPolicy{})
	repository := &memoryRepository{message: Message{ID: 12, SiteID: 5, Transport: "default", Status: StatusQueued}, claimable: true}
	transport := &countingTransport{failure: errors.New("temporary failure")}
	registry, _ := NewTransportRegistry(map[TransportAlias]Transport{"default": transport})
	worker, _ := NewWorker(5, repository, files, registry)
	payload, _ := json.Marshal(struct {
		MessageID MessageID `json:"message_id"`
	}{12})
	wrongScope := job.Envelope{ID: "01H00000000000000000000001", Name: SendJobName, ScopeID: "6", SchemaVersion: 1, Payload: payload}
	if err := worker.Handle(context.Background(), wrongScope); err == nil || !repository.claimable {
		t.Fatalf("wrong-scope delivery error = %v, claimable=%t", err, repository.claimable)
	}
	item := wrongScope
	item.ScopeID = "5"
	if err := worker.Handle(context.Background(), item); err == nil || repository.message.Status != StatusFailed || len(repository.attempts) != 1 || repository.attempts[0].Status != AttemptFailed {
		t.Fatalf("failed delivery = %v, message=%#v attempts=%#v", err, repository.message, repository.attempts)
	}
	transport.failure = nil
	if err := worker.Handle(context.Background(), item); err != nil || repository.message.Status != StatusAccepted || len(repository.attempts) != 2 || repository.attempts[1].Status != AttemptAccepted {
		t.Fatalf("retry delivery = %v, message=%#v attempts=%#v", err, repository.message, repository.attempts)
	}
}

func TestNullLogAndSMTPConfiguration(t *testing.T) {
	t.Parallel()
	result, err := (NullTransport{}).Send(context.Background(), Delivery{})
	if err != nil || result.Driver != "null" {
		t.Fatalf("null = %#v %v", result, err)
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	_, err = (LogTransport{Logger: logger}).Send(context.Background(), Delivery{Message: Message{ID: 1}})
	if err != nil || strings.Contains(logs.String(), "body") {
		t.Fatalf("log transport = %q %v", logs.String(), err)
	}
	if _, err := NewSMTPTransport(SMTPConfig{Host: "smtp.example.com", Port: 587, TLSEnabled: true, Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewSMTPTransport(SMTPConfig{Host: "", Port: 587, Timeout: time.Second}); err == nil {
		t.Fatal("invalid SMTP config accepted")
	}
}
