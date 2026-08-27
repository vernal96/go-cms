package mail

import (
	"errors"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

type TemplateID int64
type MessageID int64
type AttemptID int64
type TransportAlias string
type ContentType string
type AttachmentSource string
type MessageStatus string
type MessageOrigin string
type AttemptStatus string

const (
	ContentText ContentType = "text"
	ContentHTML ContentType = "html"

	AttachmentStatic   AttachmentSource = "static"
	AttachmentVariable AttachmentSource = "variable"

	StatusQueued   MessageStatus = "queued"
	StatusSending  MessageStatus = "sending"
	StatusAccepted MessageStatus = "accepted"
	StatusFailed   MessageStatus = "failed"

	OriginManual    MessageOrigin = "manual"
	OriginAutomatic MessageOrigin = "automatic"

	AttemptSending  AttemptStatus = "sending"
	AttemptAccepted AttemptStatus = "accepted"
	AttemptFailed   AttemptStatus = "failed"
)

var (
	ErrNotFound          = errors.New("mail item not found")
	ErrConflict          = errors.New("mail item conflict")
	ErrInvalid           = errors.New("mail item is invalid")
	ErrTemplateDisabled  = errors.New("mail template is disabled")
	ErrNoRecipients      = errors.New("mail message has no recipients")
	ErrSenderNotAllowed  = errors.New("mail sender is not allowed")
	ErrTransportNotFound = errors.New("mail transport not found")
)

type AddressTemplate struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AttachmentTemplate struct {
	Source           AttachmentSource `json:"source"`
	FileID           *file.ID         `json:"file_id,omitempty"`
	Variable         string           `json:"variable,omitempty"`
	FilenameTemplate string           `json:"filename_template,omitempty"`
}

type Template struct {
	ID          TemplateID           `json:"id"`
	SiteID      site.ID              `json:"site_id"`
	Code        string               `json:"code"`
	Name        string               `json:"name"`
	Enabled     bool                 `json:"enabled"`
	Transport   TransportAlias       `json:"transport"`
	From        AddressTemplate      `json:"from"`
	To          []AddressTemplate    `json:"to"`
	CC          []AddressTemplate    `json:"cc"`
	BCC         []AddressTemplate    `json:"bcc"`
	ReplyTo     *AddressTemplate     `json:"reply_to,omitempty"`
	Subject     string               `json:"subject"`
	ContentType ContentType          `json:"content_type"`
	TextBody    string               `json:"text_body"`
	HTMLBody    string               `json:"html_body"`
	Attachments []AttachmentTemplate `json:"attachments"`
	Variables   []field.Definition   `json:"-"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	CreatedBy   *security.UserID     `json:"created_by,omitempty"`
	UpdatedBy   *security.UserID     `json:"updated_by,omitempty"`
}

type Address struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Attachment struct {
	FileID   file.ID `json:"file_id"`
	Filename string  `json:"filename"`
	MIMEType string  `json:"mime_type"`
	Size     int64   `json:"size"`
	Checksum string  `json:"checksum_sha256"`
}

type Warning struct {
	Field    string `json:"field"`
	Variable string `json:"variable"`
	Message  string `json:"message"`
}

type RenderedMessage struct {
	From        Address      `json:"from"`
	To          []Address    `json:"to"`
	CC          []Address    `json:"cc"`
	BCC         []Address    `json:"bcc"`
	ReplyTo     *Address     `json:"reply_to,omitempty"`
	Subject     string       `json:"subject"`
	ContentType ContentType  `json:"content_type"`
	TextBody    string       `json:"text_body"`
	HTMLBody    string       `json:"html_body"`
	Attachments []Attachment `json:"attachments"`
	Warnings    []Warning    `json:"warnings"`
}

type Origin struct {
	Kind          MessageOrigin
	Source        string
	RequestedBy   *security.UserID
	RequestedName string
}

type Message struct {
	ID              MessageID        `json:"id"`
	SiteID          site.ID          `json:"site_id"`
	TemplateID      *TemplateID      `json:"template_id,omitempty"`
	TemplateCode    string           `json:"template_code"`
	TemplateName    string           `json:"template_name"`
	Transport       TransportAlias   `json:"transport"`
	RFCMessageID    string           `json:"rfc_message_id"`
	From            Address          `json:"from"`
	To              []Address        `json:"to"`
	CC              []Address        `json:"cc"`
	BCC             []Address        `json:"bcc"`
	ReplyTo         *Address         `json:"reply_to,omitempty"`
	Subject         string           `json:"subject"`
	ContentType     ContentType      `json:"content_type"`
	TextBody        string           `json:"text_body"`
	HTMLBody        string           `json:"html_body"`
	Attachments     []Attachment     `json:"attachments"`
	Status          MessageStatus    `json:"status"`
	Origin          MessageOrigin    `json:"origin"`
	OriginSource    string           `json:"origin_source"`
	RequestedAt     time.Time        `json:"requested_at"`
	RequestedBy     *security.UserID `json:"requested_by,omitempty"`
	RequestedByName string           `json:"requested_by_name"`
	AcceptedAt      *time.Time       `json:"accepted_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	AttemptCount    int              `json:"attempt_count"`
	LatestAttempt   *DeliveryAttempt `json:"latest_attempt,omitempty"`
}

type DeliveryAttempt struct {
	ID              AttemptID      `json:"id"`
	MessageID       MessageID      `json:"message_id"`
	AttemptNumber   int            `json:"attempt_number"`
	Transport       TransportAlias `json:"transport"`
	Driver          string         `json:"driver"`
	StartedAt       time.Time      `json:"started_at"`
	FinishedAt      *time.Time     `json:"finished_at,omitempty"`
	Status          AttemptStatus  `json:"status"`
	RemoteMessageID string         `json:"remote_message_id"`
	ResponseCode    string         `json:"response_code"`
	SafeError       string         `json:"safe_error"`
	CreatedAt       time.Time      `json:"created_at"`
}

type PageQuery struct {
	Page    int
	PerPage int
}

type TemplatePage struct {
	Items []Template
	Total int
}

type MessagePage struct {
	Items []Message
	Total int
}

type MessageDetail struct {
	Message  Message           `json:"message"`
	Attempts []DeliveryAttempt `json:"attempts"`
}

type QueueInput struct {
	TemplateCode string
	Values       map[string]any
	Origin       Origin
}

type ManualSendInput struct {
	TemplateID TemplateID
	Values     map[string]any
	ActorName  string
}

type DeliveryResult struct {
	Driver          string
	RemoteMessageID string
	ResponseCode    string
}
