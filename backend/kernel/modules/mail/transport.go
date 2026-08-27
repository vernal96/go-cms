package mail

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
)

type DeliveryAttachment struct {
	Attachment
	Body io.Reader
}

type Delivery struct {
	Message     Message
	Attachments []DeliveryAttachment
}

type Transport interface {
	Driver() string
	Send(context.Context, Delivery) (DeliveryResult, error)
}

type transportValidator interface{ Validate() error }

type TransportRegistry struct {
	items map[TransportAlias]Transport
}

func NewTransportRegistry(items map[TransportAlias]Transport) (*TransportRegistry, error) {
	if len(items) == 0 {
		return nil, errors.New("mail transport registry is empty")
	}
	result := &TransportRegistry{items: make(map[TransportAlias]Transport, len(items))}
	for alias, transport := range items {
		if strings.TrimSpace(string(alias)) == "" || strings.TrimSpace(string(alias)) != string(alias) || transport == nil {
			return nil, fmt.Errorf("mail transport alias %q is invalid", alias)
		}
		if validator, ok := transport.(transportValidator); ok {
			if err := validator.Validate(); err != nil {
				return nil, fmt.Errorf("mail transport %q: %w", alias, err)
			}
		}
		result.items[alias] = transport
	}
	return result, nil
}

type InvalidTransport struct{ Name string }

func (t InvalidTransport) Driver() string { return t.Name }
func (t InvalidTransport) Validate() error {
	return fmt.Errorf("unsupported mail transport driver %q", t.Name)
}
func (t InvalidTransport) Send(context.Context, Delivery) (DeliveryResult, error) {
	return DeliveryResult{}, t.Validate()
}

func (r *TransportRegistry) Transport(alias TransportAlias) (Transport, bool) {
	if r == nil {
		return nil, false
	}
	transport, exists := r.items[alias]
	return transport, exists
}

func (r *TransportRegistry) Aliases() []TransportAlias {
	result := make([]TransportAlias, 0, len(r.items))
	for alias := range r.items {
		result = append(result, alias)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

type NullTransport struct{}

func (NullTransport) Driver() string { return "null" }
func (NullTransport) Send(context.Context, Delivery) (DeliveryResult, error) {
	return DeliveryResult{Driver: "null", ResponseCode: "discarded"}, nil
}

type LogTransport struct{ Logger *slog.Logger }

func (LogTransport) Driver() string { return "log" }
func (t LogTransport) Send(ctx context.Context, delivery Delivery) (DeliveryResult, error) {
	logger := t.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "development mail accepted",
		slog.String("event", "mail.log.accepted"),
		slog.Int64("message_id", int64(delivery.Message.ID)),
		slog.String("rfc_message_id", delivery.Message.RFCMessageID),
		slog.Int("recipient_count", len(delivery.Message.To)+len(delivery.Message.CC)+len(delivery.Message.BCC)),
		slog.Int("attachment_count", len(delivery.Attachments)),
	)
	return DeliveryResult{Driver: "log", ResponseCode: "logged"}, nil
}
