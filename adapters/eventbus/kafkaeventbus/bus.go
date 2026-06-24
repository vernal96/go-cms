package kafkaeventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/vernal96/go-cms/core"
)

type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

type Bus struct {
	writer *kafka.Writer
	reader *kafka.Reader

	mu       sync.RWMutex
	handlers map[core.EventName][]core.EventHandler
}

func NewBus(config Config) (*Bus, error) {
	if len(config.Brokers) == 0 {
		return nil, errors.New("kafka brokers are empty")
	}
	for _, broker := range config.Brokers {
		if broker == "" {
			return nil, errors.New("kafka broker is empty")
		}
	}
	if config.Topic == "" {
		return nil, errors.New("kafka topic is empty")
	}

	return &Bus{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(config.Brokers...),
			Topic:    config.Topic,
			Balancer: &kafka.LeastBytes{},
		},
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: config.Brokers,
			Topic:   config.Topic,
			GroupID: config.GroupID,
		}),
		handlers: make(map[core.EventName][]core.EventHandler),
	}, nil
}

func (b *Bus) Publish(ctx context.Context, event core.Event) error {
	if event.Name == "" {
		return errors.New("event name is empty")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal kafka event %q: %w", event.Name, err)
	}

	if err := b.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Name),
		Value: payload,
	}); err != nil {
		return fmt.Errorf("publish kafka event %q: %w", event.Name, err)
	}

	return nil
}

func (b *Bus) Subscribe(name core.EventName, handler core.EventHandler) error {
	if name == "" {
		return errors.New("event name is empty")
	}
	if handler == nil {
		return errors.New("event handler is nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[name] = append(b.handlers[name], handler)

	return nil
}

func (b *Bus) Run(ctx context.Context) error {
	for {
		message, err := b.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}

			return fmt.Errorf("read kafka event: %w", err)
		}

		var event core.Event
		if err := json.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("unmarshal kafka event: %w", err)
		}
		if event.Name == "" {
			event.Name = core.EventName(message.Key)
		}

		b.mu.RLock()
		handlers := append([]core.EventHandler(nil), b.handlers[event.Name]...)
		b.mu.RUnlock()

		for _, handler := range handlers {
			if err := handler(ctx, event); err != nil {
				return fmt.Errorf("handle kafka event %q: %w", event.Name, err)
			}
		}
	}
}

func (b *Bus) Close() error {
	writerErr := b.writer.Close()
	readerErr := b.reader.Close()

	return errors.Join(writerErr, readerErr)
}

var _ core.EventBus = (*Bus)(nil)
