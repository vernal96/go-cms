package core

import "context"

type EventName string

type Event struct {
	Name    EventName
	Payload any
}

type EventHandler func(ctx context.Context, event Event) error

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(name EventName, handler EventHandler) error
}
