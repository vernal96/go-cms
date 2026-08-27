package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/messageid"
)

var ErrLeaseLost = errors.New("outbox message lease is unavailable")

type Record struct {
	MessageID    messageid.ID
	Topic        string
	Key          []byte
	Body         []byte
	Headers      map[string][]byte
	CreatedAt    time.Time
	AvailableAt  time.Time
	AttemptCount int64
	LastError    string
	LeaseOwner   string
	LeaseUntil   *time.Time
	PublishedAt  *time.Time
}

func (r Record) Message() eventbus.Message {
	headers := make(map[string][]byte, len(r.Headers))
	for key, value := range r.Headers {
		headers[key] = append([]byte(nil), value...)
	}
	return eventbus.Message{Topic: r.Topic, Key: append([]byte(nil), r.Key...), Body: append([]byte(nil), r.Body...), Headers: headers}
}

type Claim struct {
	Owner         string
	Now           time.Time
	LeaseDuration time.Duration
	Limit         int
}

type Source interface {
	Name() string
	Claim(context.Context, Claim) ([]Record, error)
	MarkPublished(context.Context, messageid.ID, string, time.Time) error
	MarkFailed(context.Context, messageid.ID, string, string, time.Time) error
	CleanupPublished(context.Context, time.Time, int) (int64, error)
}

type Provider interface {
	OutboxSources() []Source
}

type PublisherConfig struct {
	PollInterval       time.Duration
	BatchSize          int
	LeaseDuration      time.Duration
	InitialRetryDelay  time.Duration
	MaximumRetryDelay  time.Duration
	PublishedRetention time.Duration
	CleanupInterval    time.Duration
}

func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		PollInterval: 500 * time.Millisecond, BatchSize: 100, LeaseDuration: 30 * time.Second,
		InitialRetryDelay: time.Second, MaximumRetryDelay: time.Hour,
		PublishedRetention: 7 * 24 * time.Hour, CleanupInterval: time.Hour,
	}
}

func NormalizePublisherConfig(config PublisherConfig) (PublisherConfig, error) {
	defaults := DefaultPublisherConfig()
	if config.PollInterval == 0 {
		config.PollInterval = defaults.PollInterval
	}
	if config.BatchSize == 0 {
		config.BatchSize = defaults.BatchSize
	}
	if config.LeaseDuration == 0 {
		config.LeaseDuration = defaults.LeaseDuration
	}
	if config.InitialRetryDelay == 0 {
		config.InitialRetryDelay = defaults.InitialRetryDelay
	}
	if config.MaximumRetryDelay == 0 {
		config.MaximumRetryDelay = defaults.MaximumRetryDelay
	}
	if config.PublishedRetention == 0 {
		config.PublishedRetention = defaults.PublishedRetention
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = defaults.CleanupInterval
	}
	if config.PollInterval < 0 || config.BatchSize < 1 || config.LeaseDuration <= 0 ||
		config.InitialRetryDelay <= 0 || config.MaximumRetryDelay < config.InitialRetryDelay ||
		config.PublishedRetention < 0 || config.CleanupInterval <= 0 {
		return PublisherConfig{}, errors.New("outbox publisher configuration is invalid")
	}
	return config, nil
}

type Publisher struct {
	bus     eventbus.Bus
	sources []Source
	logger  *slog.Logger
	config  PublisherConfig
	owner   string
	now     func() time.Time
}

func NewPublisher(bus eventbus.Bus, sources []Source, logger *slog.Logger, config PublisherConfig) (*Publisher, error) {
	if bus == nil {
		return nil, errors.New("outbox publisher event bus is nil")
	}
	config, err := NormalizePublisherConfig(config)
	if err != nil {
		return nil, err
	}
	id, err := messageid.New()
	if err != nil {
		return nil, fmt.Errorf("create outbox publisher identity: %w", err)
	}
	seen := make(map[string]struct{}, len(sources))
	cloned := make([]Source, len(sources))
	for index, source := range sources {
		if source == nil {
			return nil, fmt.Errorf("outbox source at index %d is nil", index)
		}
		name := strings.TrimSpace(source.Name())
		if name == "" {
			return nil, fmt.Errorf("outbox source at index %d has empty name", index)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("outbox source %q is duplicated", name)
		}
		seen[name] = struct{}{}
		cloned[index] = source
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Publisher{bus: bus, sources: cloned, logger: logger, config: config, owner: string(id), now: time.Now}, nil
}

func (p *Publisher) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("outbox publisher context is nil")
	}
	p.logger.InfoContext(ctx, "outbox publisher started", slog.String("event", "outbox.publisher.started"), slog.String("publisher", p.owner))
	defer p.logger.Info("outbox publisher stopped", slog.String("event", "outbox.publisher.stopped"), slog.String("publisher", p.owner))
	cleanupAt := p.now().UTC()
	for {
		p.process(ctx)
		now := p.now().UTC()
		if !now.Before(cleanupAt) {
			p.cleanup(ctx, now)
			cleanupAt = now.Add(p.config.CleanupInterval)
		}
		timer := time.NewTimer(p.config.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (p *Publisher) process(ctx context.Context) {
	for _, source := range p.sources {
		if ctx.Err() != nil {
			return
		}
		now := p.now().UTC()
		records, err := source.Claim(ctx, Claim{Owner: p.owner, Now: now, LeaseDuration: p.config.LeaseDuration, Limit: p.config.BatchSize})
		if err != nil {
			if ctx.Err() == nil {
				p.logger.ErrorContext(ctx, "outbox claim failed", slog.String("event", "outbox.claim.failed"), slog.String("source", source.Name()), slog.Any("error", err))
			}
			continue
		}
		for _, record := range records {
			if ctx.Err() != nil {
				return
			}
			err := p.bus.Publish(ctx, record.Message())
			if err == nil {
				err = source.MarkPublished(ctx, record.MessageID, p.owner, p.now().UTC())
				if err != nil && ctx.Err() == nil {
					p.logger.ErrorContext(ctx, "outbox publish acknowledgement failed", slog.String("event", "outbox.mark_published.failed"), slog.String("source", source.Name()), slog.String("message_id", string(record.MessageID)), slog.String("topic", record.Topic), slog.Any("error", err))
				}
				continue
			}
			attempt := record.AttemptCount + 1
			delay := p.retryDelay(attempt)
			markErr := source.MarkFailed(ctx, record.MessageID, p.owner, err.Error(), p.now().UTC().Add(delay))
			p.logger.ErrorContext(ctx, "outbox publish failed", slog.String("event", "outbox.publish.failed"), slog.String("source", source.Name()), slog.String("message_id", string(record.MessageID)), slog.String("topic", record.Topic), slog.Int64("attempt", attempt), slog.Any("error", err))
			if markErr != nil && ctx.Err() == nil {
				p.logger.ErrorContext(ctx, "outbox retry persistence failed", slog.String("event", "outbox.mark_failed.failed"), slog.String("source", source.Name()), slog.String("message_id", string(record.MessageID)), slog.Any("error", markErr))
			}
		}
	}
}

func (p *Publisher) retryDelay(attempt int64) time.Duration {
	exponent := float64(attempt - 1)
	if exponent > 62 {
		exponent = 62
	}
	delay := float64(p.config.InitialRetryDelay) * math.Pow(2, exponent)
	if delay >= float64(p.config.MaximumRetryDelay) {
		return p.config.MaximumRetryDelay
	}
	return time.Duration(delay)
}

func (p *Publisher) cleanup(ctx context.Context, now time.Time) {
	before := now.Add(-p.config.PublishedRetention)
	for _, source := range p.sources {
		if ctx.Err() != nil {
			return
		}
		if _, err := source.CleanupPublished(ctx, before, p.config.BatchSize); err != nil && ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "outbox cleanup failed", slog.String("event", "outbox.cleanup.failed"), slog.String("source", source.Name()), slog.Any("error", err))
		}
	}
}
