package mail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/job"
	"github.com/vernal96/go-cms/kernel/messageid"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

var (
	TemplateReadPermission   = permission.MustCode("mail", "template", permission.Read)
	TemplateCreatePermission = permission.MustCode("mail", "template", permission.Create)
	TemplateUpdatePermission = permission.MustCode("mail", "template", permission.Update)
	TemplateDeletePermission = permission.MustCode("mail", "template", permission.Delete)
	MessageReadPermission    = permission.MustCode("mail", "message", permission.Read)
	MessageCreatePermission  = permission.MustCode("mail", "message", permission.Create)
	MessageDeletePermission  = permission.MustCode("mail", "message", permission.Delete)
)

const SendJobName = "mail.send"

type Service struct {
	siteID           site.ID
	repository       Repository
	renderer         *Renderer
	authorizer       security.Authorizer
	transports       *TransportRegistry
	defaultTransport TransportAlias
	messageIDDomain  string
}

func NewService(siteID site.ID, repository Repository, renderer *Renderer, authorizer security.Authorizer, transports *TransportRegistry, defaultTransport TransportAlias, messageIDDomain string) (*Service, error) {
	if siteID <= 0 {
		return nil, errors.New("mail service site ID is invalid")
	}
	if repository == nil || renderer == nil || authorizer == nil || transports == nil {
		return nil, errors.New("mail service dependencies are nil")
	}
	if strings.TrimSpace(string(defaultTransport)) == "" {
		defaultTransport = "default"
	}
	if _, exists := transports.Transport(defaultTransport); !exists {
		return nil, fmt.Errorf("default mail transport %q is unavailable", defaultTransport)
	}
	messageIDDomain = strings.TrimSpace(messageIDDomain)
	if messageIDDomain == "" {
		messageIDDomain = "localhost"
	}
	return &Service{siteID: siteID, repository: repository, renderer: renderer, authorizer: authorizer, transports: transports, defaultTransport: defaultTransport, messageIDDomain: messageIDDomain}, nil
}

func (s *Service) ListTemplates(ctx context.Context, actor security.Actor, query PageQuery) (TemplatePage, error) {
	if err := s.authorizer.Check(ctx, actor, TemplateReadPermission); err != nil {
		return TemplatePage{}, err
	}
	query, err := normalizePage(query)
	if err != nil {
		return TemplatePage{}, err
	}
	return s.repository.ListTemplates(ctx, s.siteID, query)
}

func (s *Service) Template(ctx context.Context, actor security.Actor, id TemplateID) (Template, error) {
	if err := s.authorizer.Check(ctx, actor, TemplateReadPermission); err != nil {
		return Template{}, err
	}
	return s.repository.TemplateByID(ctx, s.siteID, id)
}

func (s *Service) SendTemplates(ctx context.Context, actor security.Actor, query PageQuery) (TemplatePage, error) {
	if err := s.authorizer.Check(ctx, actor, MessageCreatePermission); err != nil {
		return TemplatePage{}, err
	}
	query, err := normalizePage(query)
	if err != nil {
		return TemplatePage{}, err
	}
	return s.repository.ListEnabledTemplates(ctx, s.siteID, query)
}

func (s *Service) CreateTemplate(ctx context.Context, actor security.Actor, template Template) (Template, error) {
	if err := s.authorizer.Check(ctx, actor, TemplateCreatePermission); err != nil {
		return Template{}, err
	}
	template.ID, template.SiteID = 0, s.siteID
	if template.Transport == "" {
		template.Transport = s.defaultTransport
	}
	if _, exists := s.transports.Transport(template.Transport); !exists {
		return Template{}, fmt.Errorf("%w: transport alias %q is unavailable", ErrInvalid, template.Transport)
	}
	if err := s.renderer.ValidateTemplate(template); err != nil {
		return Template{}, err
	}
	template.CreatedBy, template.UpdatedBy = actor.AuditUserID(), actor.AuditUserID()
	return s.repository.CreateTemplate(ctx, template)
}

func (s *Service) UpdateTemplate(ctx context.Context, actor security.Actor, template Template) (Template, error) {
	if err := s.authorizer.Check(ctx, actor, TemplateUpdatePermission); err != nil {
		return Template{}, err
	}
	if template.ID <= 0 {
		return Template{}, fmt.Errorf("%w: template ID is invalid", ErrInvalid)
	}
	template.SiteID = s.siteID
	if template.Transport == "" {
		template.Transport = s.defaultTransport
	}
	if _, exists := s.transports.Transport(template.Transport); !exists {
		return Template{}, fmt.Errorf("%w: transport alias %q is unavailable", ErrInvalid, template.Transport)
	}
	if err := s.renderer.ValidateTemplate(template); err != nil {
		return Template{}, err
	}
	template.UpdatedBy = actor.AuditUserID()
	return s.repository.UpdateTemplate(ctx, template)
}

func (s *Service) DeleteTemplate(ctx context.Context, actor security.Actor, id TemplateID) error {
	if err := s.authorizer.Check(ctx, actor, TemplateDeletePermission); err != nil {
		return err
	}
	return s.repository.DeleteTemplate(ctx, s.siteID, id)
}

func (s *Service) Preview(ctx context.Context, actor security.Actor, templateID TemplateID, values map[string]any) (RenderedMessage, error) {
	if err := s.authorizer.Check(ctx, actor, MessageCreatePermission); err != nil {
		return RenderedMessage{}, err
	}
	template, err := s.repository.TemplateByID(ctx, s.siteID, templateID)
	if err != nil {
		return RenderedMessage{}, err
	}
	if !template.Enabled {
		return RenderedMessage{}, ErrTemplateDisabled
	}
	return s.renderer.Render(ctx, template, values)
}

func (s *Service) QueueManual(ctx context.Context, actor security.Actor, input ManualSendInput) (Message, error) {
	if err := s.authorizer.Check(ctx, actor, MessageCreatePermission); err != nil {
		return Message{}, err
	}
	template, err := s.repository.TemplateByID(ctx, s.siteID, input.TemplateID)
	if err != nil {
		return Message{}, err
	}
	return s.queue(ctx, template, input.Values, Origin{Kind: OriginManual, RequestedBy: actor.AuditUserID(), RequestedName: strings.TrimSpace(input.ActorName)})
}

// QueueByCode is the stable programmatic integration surface for other modules.
// Infrastructure, database IDs and broker details remain internal to Mail.
func (s *Service) QueueByCode(ctx context.Context, input QueueInput) (Message, error) {
	template, err := s.repository.TemplateByCode(ctx, s.siteID, input.TemplateCode)
	if err != nil {
		return Message{}, err
	}
	if input.Origin.Kind == "" {
		input.Origin.Kind = OriginAutomatic
	}
	return s.queue(ctx, template, input.Values, input.Origin)
}

func (s *Service) queue(ctx context.Context, template Template, values map[string]any, origin Origin) (Message, error) {
	if !template.Enabled {
		return Message{}, ErrTemplateDisabled
	}
	rendered, err := s.renderer.Render(ctx, template, values)
	if err != nil {
		return Message{}, err
	}
	id, err := messageid.New()
	if err != nil {
		return Message{}, fmt.Errorf("create mail message identity: %w", err)
	}
	templateID := template.ID
	transport := template.Transport
	if transport == "" {
		transport = s.defaultTransport
	}
	if _, exists := s.transports.Transport(transport); !exists {
		return Message{}, fmt.Errorf("%w: %s", ErrTransportNotFound, transport)
	}
	now := time.Now().UTC()
	message := Message{
		SiteID: s.siteID, TemplateID: &templateID, TemplateCode: template.Code, TemplateName: template.Name,
		Transport: transport, RFCMessageID: fmt.Sprintf("<%s@%s>", id, s.messageIDDomain),
		From: rendered.From, To: rendered.To, CC: rendered.CC, BCC: rendered.BCC, ReplyTo: rendered.ReplyTo,
		Subject: rendered.Subject, ContentType: rendered.ContentType, TextBody: rendered.TextBody, HTMLBody: rendered.HTMLBody,
		Attachments: rendered.Attachments, Status: StatusQueued, Origin: origin.Kind, OriginSource: strings.TrimSpace(origin.Source),
		RequestedAt: now, RequestedBy: origin.RequestedBy, RequestedByName: strings.TrimSpace(origin.RequestedName),
	}
	item, err := job.NewScoped(SendJobName, 1, fmt.Sprint(s.siteID), struct {
		MessageID MessageID `json:"message_id"`
	}{MessageID: 0})
	if err != nil {
		return Message{}, err
	}
	// The adapter allocates the message ID and replaces the zero payload ID in
	// the same transaction before inserting the outbox row.
	body, err := json.Marshal(item)
	if err != nil {
		return Message{}, err
	}
	queued, err := s.repository.CreateMessageAndJob(ctx, message, eventbus.Message{
		Topic: job.Topic(SendJobName), Key: []byte(item.ID), Body: body,
		Headers: map[string][]byte{"content-type": []byte("application/json"), "x-cms-message-id": []byte(item.ID), "x-cms-job-name": []byte(SendJobName)},
	})
	if err != nil {
		return Message{}, err
	}
	return queued, nil
}

func (s *Service) ListMessages(ctx context.Context, actor security.Actor, query PageQuery) (MessagePage, error) {
	if err := s.authorizer.Check(ctx, actor, MessageReadPermission); err != nil {
		return MessagePage{}, err
	}
	query, err := normalizePage(query)
	if err != nil {
		return MessagePage{}, err
	}
	return s.repository.ListMessages(ctx, s.siteID, query)
}

func (s *Service) Message(ctx context.Context, actor security.Actor, id MessageID) (MessageDetail, error) {
	if err := s.authorizer.Check(ctx, actor, MessageReadPermission); err != nil {
		return MessageDetail{}, err
	}
	return s.repository.MessageDetail(ctx, s.siteID, id)
}

func (s *Service) DeleteMessage(ctx context.Context, actor security.Actor, id MessageID) error {
	if err := s.authorizer.Check(ctx, actor, MessageDeletePermission); err != nil {
		return err
	}
	return s.repository.DeleteMessage(ctx, s.siteID, id)
}

func normalizePage(query PageQuery) (PageQuery, error) {
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PerPage == 0 {
		query.PerPage = 20
	}
	if query.Page < 1 || query.PerPage < 1 || query.PerPage > 100 {
		return PageQuery{}, fmt.Errorf("%w: pagination is invalid", ErrInvalid)
	}
	return query, nil
}
